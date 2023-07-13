package postgres

import (
	"testing"
	"time"

	"github.com/ess/testscope"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func TestCallService_Get(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	service := NewCallService(testdb)

	guid, _ := core.GenGUID()

	t.Run("when the call does not exist", func(t *testing.T) {
		Cleaner.Acquire("calls")
		WithTransaction(testdb, func(tx Transaction) error {
			tx.Exec("DELETE FROM calls WHERE guid = $1", guid)
			return nil
		})

		actual, err := service.Get(guid)

		t.Run("it is nil", func(t *testing.T) {
			if actual != nil {
				t.Errorf("expected no call, got %s", actual.GUID)
			}
		})

		t.Run("it returns an error", func(t *testing.T) {
			if err == nil {
				t.Errorf("expected an error")
			}
		})

		Cleaner.Clean("calls")

	})

	t.Run("when the call exists", func(t *testing.T) {
		Cleaner.Acquire("calls")

		expected := dummyCall(&core.Call{GUID: guid})

		actual, err := service.Get(guid)

		t.Run("it is the expected call", func(t *testing.T) {
			if actual == nil {
				t.Errorf("expected a call, got none")
			}

			if actual.GUID != expected.GUID {
				t.Errorf("expected call '%s', got call '%s'", expected.GUID, actual.GUID)
			}
		})

		t.Run("it returns no error", func(t *testing.T) {
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
		})

		Cleaner.Clean("calls")
	})
}

func TestCallService_Persist(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	service := NewCallService(testdb)

	now := time.Now().UTC()

	name := "my-awesome-call"

	candidate := &core.Call{
		Name:       name,
		URL:        "http://example.com",
		AuthHeader: "whatever",
		CreatedAt:  now,
		UpdatedAt:  now,
		AppGUID:    "lolappguid",
		SpaceGUID:  "lolspaceguid",
	}

	t.Run("when the call does not exist", func(t *testing.T) {
		Cleaner.Acquire("calls")
		WithTransaction(testdb, func(tx Transaction) error {
			tx.Exec("DELETE FROM calls")
			return nil
		})

		actual, err := service.Persist(candidate)

		t.Run("it is the fully populated call", func(t *testing.T) {
			if actual == nil {
				t.Errorf("expected a populated call")
			}

			if len(actual.GUID) == 0 {
				t.Errorf("expected the call to have a GUID")
			}
		})

		t.Run("it is saved to the database", func(t *testing.T) {
			guid := actual.GUID

			got, gErr := service.Get(guid)

			if got == nil {
				t.Errorf("expected to receive a call from the db")
			}

			if gErr != nil {
				t.Errorf("expected no error from the db, got %s", gErr.Error())
			}
		})

		t.Run("it returns no error", func(t *testing.T) {
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
		})

		Cleaner.Clean("calls")

	})

	t.Run("when a call with that name already exists", func(t *testing.T) {
		Cleaner.Acquire("calls")

		guid, _ := core.GenGUID()
		dummyCall(&core.Call{GUID: guid, Name: name})

		actual, err := service.Persist(candidate)

		t.Run("it is nil", func(t *testing.T) {
			if actual != nil {
				t.Errorf("expected no call, got call %s", actual.GUID)
			}
		})

		t.Run("it returns an error", func(t *testing.T) {
			if err == nil {
				t.Errorf("expected an error")
			}
		})

		Cleaner.Clean("calls")
	})
}

func TestCallService_Delete(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	service := NewCallService(testdb)

	guid, _ := core.GenGUID()

	t.Run("when the call does not exist", func(t *testing.T) {
		Cleaner.Acquire("calls")
		WithTransaction(testdb, func(tx Transaction) error {
			tx.Exec("DELETE FROM calls WHERE guid = $1", guid)
			return nil
		})

		err := service.Delete(&core.Call{GUID: guid})

		t.Run("it returns no error", func(t *testing.T) {
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
		})

		Cleaner.Clean("calls")

	})

	t.Run("when the call exists", func(t *testing.T) {
		Cleaner.Acquire("calls")

		expected := dummyCall(&core.Call{GUID: guid})

		err := service.Delete(expected)

		t.Run("it deletes the call in question", func(t *testing.T) {
			highlander, hErr := service.Get(guid)

			if highlander != nil {
				t.Errorf("expected call %s to have been deleted", highlander.GUID)
			}

			if hErr == nil {
				t.Errorf("expected an error")
			}
		})

		t.Run("it returns no error", func(t *testing.T) {
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
		})

		Cleaner.Clean("calls")
	})
}

func TestCallService_Named(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	service := NewCallService(testdb)

	name := "james"

	t.Run("when the call does not exist", func(t *testing.T) {
		Cleaner.Acquire("calls")
		WithTransaction(testdb, func(tx Transaction) error {
			tx.Exec("DELETE FROM calls WHERE name = $1", name)
			return nil
		})

		actual, err := service.Named(name)

		t.Run("it is nil", func(t *testing.T) {
			if actual != nil {
				t.Errorf("expected no call, got %s", actual.GUID)
			}
		})

		t.Run("it returns an error", func(t *testing.T) {
			if err == nil {
				t.Errorf("expected an error")
			}
		})

		Cleaner.Clean("calls")

	})

	t.Run("when the call exists", func(t *testing.T) {
		Cleaner.Acquire("calls")

		expected := dummyCall(&core.Call{Name: name})

		actual, err := service.Named(name)

		t.Run("it is the expected call", func(t *testing.T) {
			if actual == nil {
				t.Errorf("expected a call, got none")
			}

			if actual.GUID != expected.GUID {
				t.Errorf("expected call %s, got call %s", expected.GUID, actual.GUID)
			}
		})

		t.Run("it returns no error", func(t *testing.T) {
			if err != nil {
				t.Errorf("expected no error, got %s", err.Error())
			}
		})

		Cleaner.Clean("calls")
	})
}

func TestCallService_InSpace(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	service := NewCallService(testdb)

	spaceGUID, _ := core.GenGUID()

	t.Run("when there are no matching calls", func(t *testing.T) {
		Cleaner.Acquire("calls")
		WithTransaction(testdb, func(tx Transaction) error {
			tx.Exec("DELETE FROM calls WHERE space_guid = $1", spaceGUID)
			return nil
		})

		actual := service.InSpace(spaceGUID)

		t.Run("it is an empty call collection", func(t *testing.T) {
			if len(actual) != 0 {
				t.Errorf("expected no calls, got %d", len(actual))
			}
		})

		Cleaner.Clean("calls")

	})

	t.Run("when there are matching calls", func(t *testing.T) {
		Cleaner.Acquire("calls")

		expected := make([]*core.Call, 0)
		expected = append(expected, dummyCall(&core.Call{SpaceGUID: spaceGUID}))
		expected = append(expected, dummyCall(&core.Call{SpaceGUID: spaceGUID}))
		expected = append(expected, dummyCall(&core.Call{SpaceGUID: spaceGUID}))
		expected = append(expected, dummyCall(&core.Call{SpaceGUID: spaceGUID}))
		unGUID, _ := core.GenGUID()
		unexpected := dummyCall(&core.Call{SpaceGUID: unGUID})

		actual := service.InSpace(spaceGUID)

		t.Run("it contains the proper calls", func(t *testing.T) {
			if len(actual) != len(expected) {
				t.Errorf("expected %d calls, got %d", len(expected), len(actual))
			}

			for _, call := range expected {
				found := false

				for _, acall := range actual {
					if acall.GUID == call.GUID {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("expected to see call %s", call.GUID)
				}
			}
		})

		t.Run("it omits calls that do not match", func(t *testing.T) {
			found := false
			for _, call := range actual {
				if call.GUID == unexpected.GUID {
					found = true
					break
				}
			}

			if found {
				t.Errorf("expected not to find call %s", unexpected.GUID)
			}
		})

		Cleaner.Clean("calls")
	})
}

func blank(candidate string) bool {
	return len(candidate) == 0
}

func dummyCall(call *core.Call) *core.Call {
	if call == nil {
		call = &core.Call{}
	}

	now := time.Now().UTC()
	call.CreatedAt = now
	call.UpdatedAt = now

	if blank(call.GUID) {
		call.GUID, _ = core.GenGUID()
	}

	if blank(call.Name) {
		call.Name = "dummy-call"
	}

	if blank(call.URL) {
		call.URL = "http://example.com"
	}

	if blank(call.AuthHeader) {
		call.AuthHeader = "dummy"
	}

	if blank(call.AppGUID) {
		call.AppGUID, _ = core.GenGUID()
	}

	if blank(call.SpaceGUID) {
		call.SpaceGUID, _ = core.GenGUID()
	}

	WithTransaction(testdb, func(tx Transaction) error {
		tx.Exec(
			"INSERT INTO calls VALUES($1, $2, $3, $4, $5, $6, $7, $8)",
			call.GUID,
			call.Name,
			call.URL,
			call.AuthHeader,
			call.AppGUID,
			call.SpaceGUID,
			call.CreatedAt,
			call.UpdatedAt,
		)

		return nil
	})

	return call
}
