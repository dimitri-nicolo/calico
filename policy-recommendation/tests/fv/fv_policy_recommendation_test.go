// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package fv

// TODO(dimitrin): Add tests for the following:
// - Test updates to recommendations (staged network policies), reflecting UI updates
// 		- StagedActionLean -> StageActionSet
// 		- StagedActionLean -> StageActionIgnore
// 		- StagedActionSet -> StageActionIgnore
// - Test deletes to recommendations
// - Test enforcement of recommendations (staged network policies). The recommendation should be
// ignored, once enforced. https://tigera.atlassian.net/browse/EV-4664
