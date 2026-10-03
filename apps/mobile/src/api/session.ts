type RefreshHandler = () => Promise<boolean>;

let refreshHandler: RefreshHandler | null = null;
let inflight: Promise<boolean> | null = null;

/** The auth provider registers how to renew the session; the HTTP client calls it on 401. */
export function setRefreshHandler(handler: RefreshHandler | null): void {
  refreshHandler = handler;
}

/** Single-flight refresh: concurrent 401s share one network call. */
export function refreshAccessToken(): Promise<boolean> {
  if (!refreshHandler) {
    return Promise.resolve(false);
  }
  if (!inflight) {
    inflight = refreshHandler().finally(() => {
      inflight = null;
    });
  }
  return inflight;
}
