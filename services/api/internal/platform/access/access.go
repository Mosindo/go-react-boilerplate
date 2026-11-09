// Package access centralises the rule deciding whether one user may see another user's profile or photos.
package access

import "strings"

const template = `(
  {V}::uuid = {T}::uuid
  OR (
    NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = {V}::uuid AND b.blocked_id = {T}::uuid) OR (b.blocker_id = {T}::uuid AND b.blocked_id = {V}::uuid))
    AND EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = {T}::uuid)
    AND (
      EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = {T}::uuid AND p.is_visible)
      OR EXISTS (SELECT 1 FROM matches m WHERE m.unmatched_at IS NULL
                 AND m.user_a_id = LEAST({V}::uuid, {T}::uuid) AND m.user_b_id = GREATEST({V}::uuid, {T}::uuid))
    )
  )
)`

// CanView returns a boolean SQL expression. viewer and target are SQL expressions
// (a bind parameter such as "$1" or a column such as "ph.user_id").
// A viewer can see a target when it is themselves, or when there is no block in either
// direction, the target has a profile, and the target is visible or already matched with the viewer.
func CanView(viewer, target string) string {
	return strings.NewReplacer("{V}", viewer, "{T}", target).Replace(template)
}
