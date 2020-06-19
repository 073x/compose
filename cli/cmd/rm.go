/*
   Copyright 2020 Docker, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/docker/api/client"
	"github.com/docker/api/multierror"
)

type rmOpts struct {
	force bool
}

// RmCommand deletes containers
func RmCommand() *cobra.Command {
	var opts rmOpts
	cmd := &cobra.Command{
		Use:     "rm",
		Aliases: []string{"delete"},
		Short:   "Remove containers",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(cmd.Context(), args, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force removal")

	return cmd
}

func runRm(ctx context.Context, args []string, opts rmOpts) error {
	c, err := client.New(ctx)
	if err != nil {
		return errors.Wrap(err, "cannot connect to backend")
	}

	var errs *multierror.Error
	for _, id := range args {
		err := c.ContainerService().Delete(ctx, id, opts.force)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		fmt.Println(id)
	}

	return errs.ErrorOrNil()
}
