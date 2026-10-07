import type { ReportReason } from "../types";

export const REPORT_REASONS: ReadonlyArray<{ value: ReportReason; label: string }> = [
  { value: "spam", label: "Spam or scam" },
  { value: "fake_profile", label: "Fake profile" },
  { value: "harassment", label: "Harassment or abuse" },
  { value: "inappropriate_content", label: "Inappropriate content" },
  { value: "underage", label: "Appears to be under 18" },
  { value: "other", label: "Something else" }
];

export const MAX_REPORT_DETAILS = 1000;
