import type { Gender, ReportReason } from "../api/types";

export const GENDER_LABEL: Record<Gender, string> = {
  man: "Man",
  woman: "Woman",
  non_binary: "Non-binary"
};

export const INTERESTED_IN_LABEL: Record<Gender, string> = {
  man: "Men",
  woman: "Women",
  non_binary: "Non-binary people"
};

export const REPORT_REASON_LABEL: Record<ReportReason, string> = {
  spam: "Spam or scam",
  fake_profile: "Fake profile",
  harassment: "Harassment or abuse",
  inappropriate_content: "Inappropriate content",
  underage: "Seems under 18",
  other: "Something else"
};
