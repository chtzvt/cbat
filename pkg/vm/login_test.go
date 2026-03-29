package vm_test

import (
	"strings"
	"testing"
)

func TestLoginFlow(t *testing.T) {
	// Simulate: registration creates file, then login reads it back
	r := runCBAT(t, `.header
    filename "test.bat"
.instrs
    ; registration: create user file with password
    st DESUN,"charlton"
    st DESPASS,"mypass"
    af "%DESPASS%","USERS/%DESUN%.txt"

    ; login: read file back and compare
    st user,"charlton"
    iex "USERS/%user%.txt",gooduser
    e "USER NOT FOUND"
    trm
    l gooduser
    stf PASSWD,"USERS/%user%.txt"
    st pass,"mypass"
    ieq pass,"%PASSWD%",good
    e "LOGIN FAILED: pass='%pass%' PASSWD='%PASSWD%'"
    trm
    l good
    e "LOGIN OK"
    trm`)

	if !strings.Contains(r.stdout, "LOGIN OK") {
		t.Errorf("login failed!\nOutput: %s\nVariables: %v", r.stdout, r.state["variables"])
	}
}

func TestLoginFlowWithPrompt(t *testing.T) {
	// Full flow with user input: register as "alice" with password "secret",
	// then login as "alice" with password "secret"
	r := runCBAT(t, `.header
    filename "test.bat"
.instrs
    ; registration
    stp DESUN,"Username:"
    stp DESPASS,"Password:"
    af "%DESPASS%","USERS/%DESUN%.txt"
    e "Registered %DESUN%"

    ; login
    stp user,"Login:"
    iex "USERS/%user%.txt",gooduser
    e "USER NOT FOUND"
    trm
    l gooduser
    stf PASSWD,"USERS/%user%.txt"
    stp pass,"Password:"
    ieq pass,"%PASSWD%",good
    e "FAIL: pass='%pass%' PASSWD='%PASSWD%'"
    trm
    l good
    e "Welcome %user%!"
    trm`,
		"alice",   // registration username
		"secret",  // registration password
		"alice",   // login username
		"secret",  // login password
	)

	if !strings.Contains(r.stdout, "Welcome alice!") {
		t.Errorf("login failed!\nOutput: %s", r.stdout)
	}
}
