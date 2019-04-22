package brigade

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"sync"

	"github.com/lovethedrake/prototype/pkg/config"
	"k8s.io/client-go/kubernetes"
)

var tagRefRegex = regexp.MustCompile("refs/tags/(.*)")

// Executor is the public interface for the Brigade executor
type Executor interface {
	ExecuteBuild(
		ctx context.Context,
		project Project,
		event Event,
	) error
}

type executor struct {
	kubeClient kubernetes.Interface
}

// NewExecutor returns an executor suitable for use with Brigade
func NewExecutor(kubeClient kubernetes.Interface) Executor {
	return &executor{
		kubeClient: kubeClient,
	}
}

// TODO: Implement this
func (e *executor) ExecuteBuild(
	ctx context.Context,
	project Project,
	event Event,
) error {
	var branch, tag string
	// There are really only two things we're interested in:
	//
	//   1. Check suite requested / re-requested. Github will send one of these
	//      anytime there is a push to a branch, in which case, the
	//      check_suite.head_branch field will indicate the branch that was pushed
	//      to, OR in the case of a PR, the Brigade Gtihub gateway will compell
	//      Github to forward a check suite request, in which case,
	//      check_suite.head_branch will be null (JS) or nil (Go). No branch name
	//      is a valid as SOME branch name for the purposes of determining whether
	//      some pipeline needs to be executed.
	//
	//   2. A push request whose ref field indicates it is a tag.
	//
	// Nothing else will trigger anything.
	switch event.Type {
	case "check_suite:requested", "check_suite:rerequested":
		cse := checkSuiteEvent{}
		if err := json.Unmarshal(event.Payload, &cse); err != nil {
			return err
		}
		if cse.Body.CheckSuite.HeadBranch != nil {
			branch = *cse.Body.CheckSuite.HeadBranch
		}
	case "push":
		pushEvent := pushEvent{}
		if err := json.Unmarshal(event.Payload, &pushEvent); err != nil {
			return err
		}
		refSubmatches := tagRefRegex.FindStringSubmatch(pushEvent.Ref)
		if len(refSubmatches) != 2 {
			log.Println(
				"received push event that wasn't for a new tag-- nothing to execute",
			)
			return nil
		}
		tag = refSubmatches[1]
	default:
		log.Printf(
			"received event type %s-- nothing to execute",
			event.Type,
		)
		return nil
	}
	log.Printf("branch: \"%s\"; tag: \"%s\"", branch, tag)
	config, err := config.NewConfigFromFile("/vcs/Drakefile.yaml")
	if err != nil {
		return err
	}
	pipelines := config.GetAllPipelines()
	errCh := make(chan error)
	wg := &sync.WaitGroup{}
	for _, pipeline := range pipelines {
		if meetsCriteria, err := pipeline.Matches(branch, tag); err != nil {
			return err
		} else if meetsCriteria {
			wg.Add(1)
			go e.runPipeline(project, event, pipeline, wg, errCh)
		}
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	errs := []error{}
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 1 {
		return &multiError{errs: errs}
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return nil
}
