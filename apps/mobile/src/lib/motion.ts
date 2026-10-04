import { useEffect, useState } from "react";
import { AccessibilityInfo } from "react-native";

/** Tracks the OS "reduce motion" setting. Defaults to false until known. */
export function useReducedMotion(): boolean {
  const [reduced, setReduced] = useState(false);

  useEffect(() => {
    let active = true;
    AccessibilityInfo.isReduceMotionEnabled?.()
      .then((value) => {
        if (active) setReduced(value);
      })
      .catch(() => undefined);
    const subscription = AccessibilityInfo.addEventListener?.("reduceMotionChanged", setReduced);
    return () => {
      active = false;
      subscription?.remove?.();
    };
  }, []);

  return reduced;
}
