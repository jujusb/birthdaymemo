// Guest-only mode detection for `--guest-only` backend deployments.
//
// Background: the guest backend (`GuestRoutes`) only mounts public share APIs
// (`GET /api/public/s/...`, languages/i18n) and serves `/` + static assets.
// Every other server route (login/captcha/me/admin/...) does not exist there
// and returns 404. The frontend is a hash-router SPA served from `/`, so
// without a guard it would still render login/admin pages whose APIs 404.
//
// Detection strategy (frontend-only, no backend change): probe an endpoint
// that exists in full mode but never in guest-only mode. `GET /api/captcha`
// is public, side-effect free and a good candidate:
//   - full mode  -> 200 (or another non-404, e.g. 429/500)
//   - guest-only -> 404 (route not mounted)
// The result is cached for the lifetime of the page.

let cached: boolean | null = null

export async function detectGuestOnlyMode(): Promise<boolean> {
  if (cached !== null) return cached
  try {
    const res = await fetch('/api/captcha', {
      method: 'GET',
      credentials: 'same-origin',
    })
    cached = res.status === 404
  } catch {
    // Network failure: fail open (behave like full mode). Share pages work
    // in both modes, so this never blocks legitimate share access.
    cached = false
  }
  return cached
}

export function isGuestOnlyMode(): boolean {
  return cached === true
}

export function setGuestOnlyMode(v: boolean): void {
  cached = v
}
