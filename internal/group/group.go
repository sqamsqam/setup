package group

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	setupexec "github.com/sqamsqam/setup/internal/exec"
	"github.com/sqamsqam/setup/internal/user"
)

var groupNameRe = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("group must not be empty")
	}
	if !groupNameRe.MatchString(name) {
		return fmt.Errorf("invalid group %q: must start with a lowercase letter or underscore, followed by letters, digits, hyphens, or underscores", name)
	}
	if len(name) > 32 {
		return fmt.Errorf("group too long: %d characters (max 32)", len(name))
	}
	return nil
}

func Create(runner setupexec.CmdRunner, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if exists, err := groupExists(runner, name); err != nil {
		return err
	} else if exists {
		setupexec.PrintStep(fmt.Sprintf("Group %s already exists, skipping creation", name))
		return nil
	}
	setupexec.PrintStep(fmt.Sprintf("Creating group %s", name))
	return runner.Run("groupadd", name)
}

func Delete(runner setupexec.CmdRunner, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	info, err := lookupGroup(runner, name)
	if err != nil {
		return err
	}
	users, err := primaryGroupUsers(runner, info.gid)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return fmt.Errorf("refusing to delete group %q: primary group for %s", name, strings.Join(users, ", "))
	}
	setupexec.PrintStep(fmt.Sprintf("Deleting group %s", name))
	return runner.Run("delgroup", name)
}

func List(runner setupexec.CmdRunner) ([]string, error) {
	out, err := runner.Output("getent", "group")
	if err != nil {
		return nil, err
	}
	var groups []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 1 || fields[0] == "" {
			continue
		}
		groups = append(groups, fields[0])
	}
	sort.Strings(groups)
	return groups, nil
}

func AddUser(runner setupexec.CmdRunner, username, name string) error {
	if err := user.ValidateUsername(username); err != nil {
		return err
	}
	if _, _, err := runner.LookupUser(username); err != nil {
		return err
	}
	if err := ValidateName(name); err != nil {
		return err
	}
	if _, err := lookupGroup(runner, name); err != nil {
		return err
	}
	if inGroup, err := userInGroup(runner, username, name); err != nil {
		return err
	} else if inGroup {
		setupexec.PrintStep(fmt.Sprintf("User %s is already in group %s, skipping", username, name))
		return nil
	}
	setupexec.PrintStep(fmt.Sprintf("Adding %s to group %s", username, name))
	return runner.Run("usermod", "-aG", name, username)
}

func RemoveUser(runner setupexec.CmdRunner, username, name string) error {
	if err := user.ValidateUsername(username); err != nil {
		return err
	}
	if _, _, err := runner.LookupUser(username); err != nil {
		return err
	}
	if err := ValidateName(name); err != nil {
		return err
	}
	if _, err := lookupGroup(runner, name); err != nil {
		return err
	}
	if inGroup, err := userInGroup(runner, username, name); err != nil {
		return err
	} else if !inGroup {
		setupexec.PrintStep(fmt.Sprintf("User %s is not in group %s, skipping", username, name))
		return nil
	}
	setupexec.PrintStep(fmt.Sprintf("Removing %s from group %s", username, name))
	return runner.Run("gpasswd", "-d", username, name)
}

type groupInfo struct {
	name string
	gid  int
}

func groupExists(runner setupexec.CmdRunner, name string) (bool, error) {
	out, err := runner.Output("getent", "group", name)
	if err != nil || strings.TrimSpace(out) == "" {
		return false, nil
	}
	return true, nil
}

func lookupGroup(runner setupexec.CmdRunner, name string) (groupInfo, error) {
	out, err := runner.Output("getent", "group", name)
	if err != nil {
		return groupInfo{}, fmt.Errorf("group %q does not exist", name)
	}
	if strings.TrimSpace(out) == "" && setupexec.IsDryRun(runner) {
		return groupInfo{name: name, gid: 99999}, nil
	}
	fields := strings.Split(strings.TrimSpace(out), ":")
	if len(fields) < 3 {
		return groupInfo{}, fmt.Errorf("invalid group entry for %q", name)
	}
	gid, err := strconv.Atoi(fields[2])
	if err != nil {
		return groupInfo{}, fmt.Errorf("parse gid for group %q: %w", name, err)
	}
	return groupInfo{name: fields[0], gid: gid}, nil
}

func primaryGroupUsers(runner setupexec.CmdRunner, gid int) ([]string, error) {
	out, err := runner.Output("getent", "passwd")
	if err != nil {
		return nil, err
	}
	var users []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		userGID, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}
		if userGID == gid {
			users = append(users, fields[0])
		}
	}
	sort.Strings(users)
	return users, nil
}

func userInGroup(runner setupexec.CmdRunner, username, name string) (bool, error) {
	out, err := runner.Output("id", "-nG", username)
	if err != nil {
		return false, err
	}
	for _, group := range strings.Fields(out) {
		if group == name {
			return true, nil
		}
	}
	return false, nil
}
