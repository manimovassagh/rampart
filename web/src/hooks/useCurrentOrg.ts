import { useState, useEffect } from "react";
import { getMe } from "../api/auth";
import { getOrg } from "../api/organizations";
import type { OrgResponse } from "../types";

let cachedOrg: OrgResponse | null = null;
let fetchPromise: Promise<OrgResponse | null> | null = null;

async function resolveOrg(): Promise<OrgResponse | null> {
  const me = await getMe();
  if (!me) return null;
  return getOrg(me.org_id);
}

function fetchOnce(): Promise<OrgResponse | null> {
  if (!fetchPromise) {
    fetchPromise = resolveOrg().catch(() => null);
  }
  return fetchPromise;
}

export function clearOrgCache() {
  cachedOrg = null;
  fetchPromise = null;
}

export function useCurrentOrg() {
  const [org, setOrg] = useState<OrgResponse | null>(cachedOrg);
  const [loading, setLoading] = useState(!cachedOrg);

  useEffect(() => {
    if (cachedOrg) return;
    fetchOnce().then((result) => {
      cachedOrg = result;
      setOrg(result);
      setLoading(false);
    });
  }, []);

  return { org, loading };
}
