// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package overlord

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/firstboot"
	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/overlord/snapstate"
	"github.com/snapcore/snapd/overlord/state"
	"github.com/snapcore/snapd/snap"
)

func populateStateFromInstalled() error {
	if osutil.FileExists(dirs.SnapStateFile) {
		return fmt.Errorf("cannot create state: state %q already exists", dirs.SnapStateFile)
	}

	ovld, err := New()
	if err != nil {
		return err
	}
	ovld.Loop()
	st := ovld.State()

	all, err := filepath.Glob(filepath.Join(dirs.SnapBlobDir, "*.snap"))
	if err != nil {
		return err
	}

	for _, snapPath := range all {
		sf, err := snap.Open(snapPath)
		if err != nil {
			return err
		}
		info, err := snap.ReadInfoFromSnapFile(sf, nil)
		if err != nil {
			return err
		}
		fmt.Printf("Installing %s\n", info.Name())

		st.Lock()
		ts, err := snapstate.InstallPathWithSideInfo(st, info.Name(), snapPath, "", 0)
		if err != nil {
			return err
		}

		// FIXME: make this a single change
		msg := fmt.Sprintf("First boot install of %s", filepath.Base(info.Name()))
		chg := st.NewChange("install-snap", msg)
		chg.AddAll(ts)
		st.Unlock()

		// do it and wait for ready
		st.EnsureBefore(0)
		<-chg.Ready()
		if chg.Status() != state.DoneStatus {
			return fmt.Errorf("cannot run chg: %v", chg)
		}

	}
	ovld.Stop()

	return nil
}

// FirstBoot will do some initial boot setup and then sync the
// state
func FirstBoot() error {
	if firstboot.HasRun() {
		return firstboot.ErrNotFirstBoot
	}
	defer firstboot.StampFirstBoot()
	if err := firstboot.EnableFirstEther(); err != nil {
		logger.Noticef("Failed to bring up ethernet: %s", err)
	}

	return populateStateFromInstalled()
}
