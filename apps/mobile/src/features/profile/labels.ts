import type { Gender } from "../../api/types";

export const GENDER_LABELS: Record<Gender, string> = { man: "Man", woman: "Woman", nonbinary: "Non-binary" };
export const INTERESTED_IN_LABELS: Record<Gender, string> = {
  man: "Men",
  woman: "Women",
  nonbinary: "Non-binary people"
};

export const REPORT_REASON_LABELS = {
  spam: "Spam or scam",
  fake: "Fake profile",
  harassment: "Harassment or abuse",
  inappropriate: "Inappropriate content",
  underage: "Seems under 18",
  other: "Something else"
} as const;
