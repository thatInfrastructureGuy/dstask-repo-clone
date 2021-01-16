package dstask

// main task data structures

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// when referring to tasks by ID, NON_RESOLVED_STATUSES must be loaded exclusively --
// even if the filter is set to show issues that have only some statuses.
type Query struct {
	Cmd           string
	IDs           []int
	Tags          []string
	AntiTags      []string
	Project       string
	AntiProjects  []string
	Priority      string
	Template      int
	Text          string
	IgnoreContext bool
	// any words after the note operator: /
	Note string
}

// reconstruct args string
func (query Query) String() string {
	var args []string

	for _, id := range query.IDs {
		args = append(args, strconv.Itoa(id))
	}

	for _, tag := range query.Tags {
		args = append(args, "+"+tag)
	}
	for _, tag := range query.AntiTags {
		args = append(args, "-"+tag)
	}

	if query.Project != "" {
		args = append(args, "project:"+query.Project)
	}

	for _, project := range query.AntiProjects {
		args = append(args, "-project:"+project)
	}

	if query.Priority != "" {
		args = append(args, query.Priority)
	}

	if query.Template > 0 {
		args = append(args, fmt.Sprintf("template:%v", query.Template))
	}

	if query.Text != "" {
		args = append(args, "\""+query.Text+"\"")
	}

	return strings.Join(args, " ")
}

func (query Query) PrintContextDescription() {
	var envVarNotification string
	if os.Getenv("DSTASK_CONTEXT") != "" {
		envVarNotification = " (set by DSTASK_CONTEXT)"
	}
	if query.String() != "" {
		fmt.Printf("\033[33mActive context%s: %s\033[0m\n", envVarNotification, query)
	}
}

// returns true if the query has positive or negative projects/tags,
// priorities, template
func (query Query) HasOperators() bool {
	return (
		len(query.Tags) > 0 ||
		len(query.AntiTags) > 0 ||
		query.Project != "" ||
		len(query.AntiProjects) > 0 ||
		query.Priority != "" ||
		query.Template > 0 )
}

// ParseQuery parses the raw command line typed by the user.
func ParseQuery(args ...string) Query {
	var cmd string
	var ids []int
	var tags []string
	var antiTags []string
	var project string
	var antiProjects []string
	var priority string
	var template int
	var words []string
	var notesModeActivated bool
	var notes []string
	var ignoreContext bool

	// something other than an ID has been parsed -- accept no more IDs
	var IDsExhausted bool

	for _, item := range args {
		lcItem := strings.ToLower(item)

		if notesModeActivated {
			// no more parsing syntax
			notes = append(notes, item)
			continue
		}

		if cmd == "" && StrSliceContains(ALL_CMDS, lcItem) {
			cmd = lcItem
			continue
		}

		if s, err := strconv.ParseInt(item, 10, 64); !IDsExhausted && err == nil {
			ids = append(ids, int(s))
			continue
		}

		if item == IGNORE_CONTEXT_KEYWORD {
			ignoreContext = true
		} else if item == NOTE_MODE_KEYWORD {
			notesModeActivated = true
		} else if strings.HasPrefix(lcItem, "project:") {
			project = lcItem[8:]
		} else if strings.HasPrefix(lcItem, "+project:") {
			project = lcItem[9:]
		} else if strings.HasPrefix(lcItem, "-project:") {
			antiProjects = append(antiProjects, lcItem[9:])
		} else if strings.HasPrefix(lcItem, "template:") {
			if s, err := strconv.ParseInt(lcItem[9:], 10, 64); err == nil {
				template = int(s)
			}
		} else if len(item) > 1 && lcItem[0:1] == "+" {
			tags = append(tags, lcItem[1:])
		} else if len(item) > 1 && lcItem[0:1] == "-" {
			antiTags = append(antiTags, lcItem[1:])
		} else if IsValidPriority(item) {
			priority = item
		} else {
			words = append(words, item)
		}

		IDsExhausted = true
	}

	return Query{
		Cmd:           cmd,
		IDs:           ids,
		Tags:          tags,
		AntiTags:      antiTags,
		Project:       project,
		AntiProjects:  antiProjects,
		Priority:      priority,
		Template:      template,
		Text:          strings.Join(words, " "),
		Note:          strings.Join(notes, " "),
		IgnoreContext: ignoreContext,
	}
}

// used for applying a context to a new task
func (query *Query) Merge(context Query) {
	for _, tag := range context.Tags {
		if !StrSliceContains(query.Tags, tag) {
			query.Tags = append(query.Tags, tag)
		}
	}

	for _, tag := range context.AntiTags {
		if !StrSliceContains(query.AntiTags, tag) {
			query.AntiTags = append(query.AntiTags, tag)
		}
	}

	// TODO same for antitags
	if context.Project != "" {
		if query.Project != "" && query.Project != context.Project {
			ExitFail("Could not apply context, project conflict")
		} else {
			query.Project = context.Project
		}
	}

	if context.Priority != "" {
		if query.Priority != "" {
			ExitFail("Could not apply context, priority conflict")
		} else {
			query.Priority = context.Priority
		}
	}
}
