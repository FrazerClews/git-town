Feature: git-extract handling conflicting remote main branch updates without open changes

  Background:
    Given I am on a feature branch
    And the following commits exist in my repository
      | BRANCH  | LOCATION | MESSAGE                   | FILE NAME        | FILE CONTENT   |
      | main    | remote   | conflicting remote commit | conflicting_file | remote content |
      | main    | local    | conflicting local commit  | conflicting_file | local content  |
      | feature | local    | feature commit            | feature_file     |                |
      | feature | local    | refactor commit           | refactor_file    |                |
    When I run `git extract refactor` with the last commit sha while allowing errors


  Scenario: result
    Then my repo has a rebase in progress
    And there is an abort script for "git extract"


  Scenario: aborting
    When I run `git extract --abort`
    Then I end up on my feature branch
    And there is no "refactor" branch
    And I have the following commits
      | BRANCH  | LOCATION | MESSAGE                   | FILES            |
      | main    | remote   | conflicting remote commit | conflicting_file |
      | main    | local    | conflicting local commit  | conflicting_file |
      | feature | local    | feature commit            | feature_file     |
      | feature | local    | refactor commit           | refactor_file    |
    And there is no rebase in progress
    And there is no abort script for "git extract" anymore
