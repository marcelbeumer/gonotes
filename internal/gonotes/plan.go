package gonotes

import (
	"fmt"
	"sort"
	"strings"
)

type Operation interface {
	String() string
	Error() error
}

type RebuildTagsFs struct{}

func (o *RebuildTagsFs) String() string {
	return "rebuild tags filesystem"
}

func (o *RebuildTagsFs) Error() error {
	return nil
}

type RenameNote struct {
	From       string
	To         string
	DeleteFrom bool
	Err        *string
}

func (o *RenameNote) String() string {
	return fmt.Sprintf(
		"rename note \"%s\" to \"%s (leaves orphan: %v)\"",
		o.From,
		o.To,
		o.DeleteFrom,
	)
}

func (o *RenameNote) Error() error {
	if o.Err == nil {
		return nil
	}
	return fmt.Errorf(
		"can not rename note \"%s\" to \"%s\": %s",
		o.From,
		o.To,
		*o.Err,
	)
}

type UpdateNote struct {
	Name string
	Err  *string
}

func (o *UpdateNote) String() string {
	return fmt.Sprintf("update note \"%s\"", o.Name)
}

func (o *UpdateNote) Error() error {
	if o.Err == nil {
		return nil
	}
	return fmt.Errorf(
		"can not update note \"%s\": %s",
		o.Name,
		*o.Err,
	)
}

type Plan []Operation

func (p *Plan) String() string {
	lines := make([]string, 0, len(*p)+1)
	lines = append(lines, fmt.Sprintf("%d operation(s) planned:", len(*p)))
	for _, v := range *p {
		lines = append(lines, fmt.Sprintf("- %s", v.String()))
	}
	return strings.Join(lines, "\n")
}

func (p *Plan) Errors() []error {
	var errs []error
	for _, op := range *p {
		if err := op.Error(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (p *Plan) HasErrors() bool {
	for _, op := range *p {
		if err := op.Error(); err != nil {
			return true
		}
	}
	return false
}

func (r *Repository) Plan() (Plan, error) {
	oldPaths := map[string]string{}
	oldNotes := map[string]*Note{}

	keep := map[string]struct{}{}    // [target]
	orphan := map[string]struct{}{}  // [target]
	moveTo := map[string][]string{}  // [target][]source
	moveFrom := map[string]string{}  // [target][]source
	changed := map[string]struct{}{} // [source]

	for oldName, n := range r.notes {
		oldPath := r.notePaths[oldName]
		oldPaths[oldName] = oldPath
		oldNotes[oldName] = n

		newName := NoteName(*n)
		delete(orphan, newName)

		md, err := n.Marhsal()
		if err != nil {
			return nil, err
		}

		if n.Raw != string(md) {
			changed[oldName] = struct{}{}
		}

		switch {
		case oldName != newName:
			moveTo[newName] = append(moveTo[newName], oldName)
			moveFrom[oldName] = newName
			_, keepOk := keep[oldName]
			_, moveOk := moveTo[oldName]
			if !keepOk && !moveOk {
				orphan[oldName] = struct{}{}
			}
		default:
			keep[newName] = struct{}{}
		}
	}

	namesMap := map[string]struct{}{}

	for name := range moveFrom {
		namesMap[name] = struct{}{}
	}
	for name := range changed {
		namesMap[name] = struct{}{}
	}

	var names []string
	for name := range namesMap {
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})

	plan := Plan{}

	for _, name := range names {
		_, inChanged := changed[name]
		moveTarget, inMoveFrom := moveFrom[name]

		switch {
		case inMoveFrom:
			_, isOrphan := orphan[name]
			moveToSources := moveTo[moveTarget]
			_, inKeep := keep[moveTarget]

			var errp *string
			switch {
			case len(moveToSources) > 1:
				s := fmt.Sprintf(
					"%d other notes want to rename to this name",
					len(moveToSources),
				)
				errp = &s
			case inKeep:
				s := fmt.Sprintf("another note already has this name")
				errp = &s
			}

			plan = append(plan, &RenameNote{
				From:       name,
				To:         moveTarget,
				DeleteFrom: isOrphan,
				Err:        errp,
			})

		case inChanged:
			plan = append(plan, &UpdateNote{Name: name})
		}
	}

	// Always sync tags fs.
	plan = append(plan, &RebuildTagsFs{})

	return plan, nil
}
