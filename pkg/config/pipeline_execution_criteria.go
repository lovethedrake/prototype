package config

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type pipelineExecutionCriteria struct {
	ManualOnly bool            `json:"manualOnly"`
	Whitelist  *branchSelector `json:"only"`
	Blacklist  *branchSelector `json:"ignore"`
}

type branchSelector struct {
	Branches []string `json:"branches"`
	Tags     []string `json:"tags"`
}

func (p *pipelineExecutionCriteria) Matches(branch, tag string) (bool, error) {
	if p.ManualOnly {
		return false, nil
	}
	// If there were no whitelist, whitelist is implicitly all-inclusive, so we'll
	// start by assuming we match it and look to prove that we don't.
	matchesWhitelist := true
	if p.Whitelist != nil {
		var err error
		matchesWhitelist, err = p.Whitelist.Matches(branch, tag)
		if err != nil {
			return false, err
		}
	}
	// If there's no blacklist, the blacklist is implicitly the null set, so we'll
	// start by assuming we don't match it and look to prove that we do.
	matchesBlacklist := false
	if p.Blacklist != nil {
		var err error
		matchesBlacklist, err = p.Blacklist.Matches(branch, tag)
		if err != nil {
			return false, err
		}
	}
	return matchesWhitelist && !matchesBlacklist, nil
}

func (b *branchSelector) Matches(branch, tag string) (bool, error) {
	if tag != "" {
		for _, tg := range b.Tags {
			if strings.HasPrefix(tg, "/") && strings.HasSuffix(tg, "/") {
				tg = tg[1 : len(tg)-1]
				regex, err := regexp.Compile(tg)
				if err != nil {
					return false, errors.Wrapf(
						err,
						"error compiling regular expression %s",
						tg,
					)
				}
				if regex.MatchString(tag) {
					return true, nil
				}
				continue
			}
			if tg == tag {
				return true, nil
			}
		}
		// If a tag was provided, the match is only on the basis of the tag
		return false, nil
	}
	if branch != "" {
		for _, br := range b.Branches {
			if strings.HasPrefix(br, "/") && strings.HasSuffix(br, "/") {
				br = br[1 : len(br)-1]
				regex, err := regexp.Compile(br)
				if err != nil {
					return false, errors.Wrapf(
						err,
						"error compiling regular expression %s",
						br,
					)
				}
				if regex.MatchString(branch) {
					return true, nil
				}
				continue
			}
			if br == branch {
				return true, nil
			}
		}
	}
	return false, nil
}
