package cmd

import (
	"strings"

	"github.com/Originate/git-town/src/git"
	"github.com/Originate/git-town/src/prompt"
	"github.com/Originate/git-town/src/script"
	"github.com/Originate/git-town/src/steps"
	"github.com/Originate/git-town/src/util"

	"github.com/spf13/cobra"
)

type shipConfig struct {
	BranchToShip  string
	InitialBranch string
}

var commitMessage string

var shipCmd = &cobra.Command{
	Use:   "ship",
	Short: "Deliver a completed feature branch",
	Long: `Deliver a completed feature branch

Squash-merges the current branch, or <branch_name> if given,
into the main branch, resulting in linear history on the main branch.

- syncs the main branch
- pulls remote updates for <branch_name>
- merges the main branch into <branch_name>
- squash-merges <branch_name> into the main branch
  with commit message specified by the user
- pushes the main branch to the remote repository
- deletes <branch_name> from the local and remote repositories

Only shipping of direct children of the main branch is allowed.
To ship a nested child branch, all ancestor branches have to be shipped or killed.`,
	Run: func(cmd *cobra.Command, args []string) {
		steps.Run(steps.RunOptions{
			CanSkip:              func() bool { return false },
			Command:              "ship",
			IsAbort:              abortFlag,
			IsContinue:           continueFlag,
			IsSkip:               false,
			IsUndo:               undoFlag,
			SkipMessageGenerator: func() string { return "" },
			StepListGenerator: func() steps.StepList {
				config := checkShipPreconditions(args)
				return getShipStepList(config)
			},
		})
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		err := validateMaxArgs(args, 1)
		if err != nil {
			return err
		}
		err = git.ValidateIsRepository()
		if err != nil {
			return err
		}
		prompt.EnsureIsConfigured()
		return nil
	},
}

func checkShipPreconditions(args []string) (result shipConfig) {
	result.InitialBranch = git.GetCurrentBranchName()
	if len(args) == 0 {
		result.BranchToShip = result.InitialBranch
	} else {
		result.BranchToShip = args[0]
	}
	if result.BranchToShip == result.InitialBranch {
		git.EnsureDoesNotHaveOpenChanges("Did you mean to commit them before shipping?")
	}
	if git.HasRemote("origin") {
		script.Fetch()
	}
	if result.BranchToShip != result.InitialBranch {
		git.EnsureHasBranch(result.BranchToShip)
	}
	git.EnsureIsFeatureBranch(result.BranchToShip, "Only feature branches can be shipped.")
	prompt.EnsureKnowsParentBranches([]string{result.BranchToShip})
	ensureParentBranchIsMainOrPerennialBranch(result.BranchToShip)
	return
}

func ensureParentBranchIsMainOrPerennialBranch(branchName string) {
	parentBranch := git.GetParentBranch(branchName)
	if !git.IsMainBranch(parentBranch) && !git.IsPerennialBranch(parentBranch) {
		ancestors := git.GetAncestorBranches(branchName)
		ancestorsWithoutMainOrPerennial := ancestors[1:]
		oldestAncestor := ancestorsWithoutMainOrPerennial[0]
		util.ExitWithErrorMessage(
			"Shipping this branch would ship "+strings.Join(ancestorsWithoutMainOrPerennial, ", ")+" as well.",
			"Please ship \""+oldestAncestor+"\" first.",
		)
	}
}

func getShipStepList(config shipConfig) (result steps.StepList) {
	branchToMergeInto := git.GetParentBranch(config.BranchToShip)
	isShippingInitialBranch := config.BranchToShip == config.InitialBranch
	result.AppendList(steps.GetSyncBranchSteps(branchToMergeInto))
	result.Append(&steps.CheckoutBranchStep{BranchName: config.BranchToShip})
	result.Append(&steps.MergeTrackingBranchStep{})
	result.Append(&steps.MergeBranchStep{BranchName: branchToMergeInto})
	result.Append(&steps.EnsureHasShippableChangesStep{BranchName: config.BranchToShip})
	result.Append(&steps.CheckoutBranchStep{BranchName: branchToMergeInto})
	result.Append(&steps.SquashMergeBranchStep{BranchName: config.BranchToShip, CommitMessage: commitMessage})
	if git.HasRemote("origin") {
		result.Append(&steps.PushBranchStep{BranchName: branchToMergeInto, Undoable: true})
	}
	childBranches := git.GetChildBranches(config.BranchToShip)
	if git.HasTrackingBranch(config.BranchToShip) && len(childBranches) == 0 {
		result.Append(&steps.DeleteRemoteBranchStep{BranchName: config.BranchToShip, IsTracking: true})
	}
	result.Append(&steps.DeleteLocalBranchStep{BranchName: config.BranchToShip})
	result.Append(&steps.DeleteParentBranchStep{BranchName: config.BranchToShip})
	for _, child := range childBranches {
		result.Append(&steps.SetParentBranchStep{BranchName: child, ParentBranchName: branchToMergeInto})
	}
	result.Append(&steps.DeleteAncestorBranchesStep{})
	if !isShippingInitialBranch {
		result.Append(&steps.CheckoutBranchStep{BranchName: config.InitialBranch})
	}
	result.Wrap(steps.WrapOptions{RunInGitRoot: true, StashOpenChanges: !isShippingInitialBranch})
	return
}

func init() {
	shipCmd.Flags().BoolVar(&abortFlag, "abort", false, abortFlagDescription)
	shipCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Specify the commit message for the squash commit")
	shipCmd.Flags().BoolVar(&continueFlag, "continue", false, continueFlagDescription)
	shipCmd.Flags().BoolVar(&undoFlag, "undo", false, undoFlagDescription)
	RootCmd.AddCommand(shipCmd)
}