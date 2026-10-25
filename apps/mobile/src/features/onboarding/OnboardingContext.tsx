import React, { createContext, useContext, useMemo, useState } from "react";
import type { Gender, Preferences, ProfileState } from "../../api";
import { splitIsoDate } from "../../shared/validation";
import { DEFAULT_PREFERENCES } from "../profile/components/PreferencesForm";
import type { BirthDateValue } from "../profile/components/BirthDateField";

export type OnboardingDraft = {
  firstName: string;
  birth: BirthDateValue;
  isoBirthDate: string;
  gender: Gender | null;
  bio: string;
  interests: string[];
  preferences: Preferences;
};

type OnboardingContextValue = {
  draft: OnboardingDraft;
  update: (patch: Partial<OnboardingDraft>) => void;
  /** Leaves onboarding for the main app. */
  finish: () => void;
};

const OnboardingContext = createContext<OnboardingContextValue | null>(null);

export function draftFromProfile(state: ProfileState | undefined): OnboardingDraft {
  const profile = state?.profile ?? null;
  return {
    firstName: profile?.firstName ?? "",
    birth: profile ? splitIsoDate(profile.birthDate) : { day: "", month: "", year: "" },
    isoBirthDate: profile?.birthDate ?? "",
    gender: profile?.gender ?? null,
    bio: profile?.bio ?? "",
    interests: profile?.interests.map((interest) => interest.slug) ?? [],
    preferences: state?.preferences ?? DEFAULT_PREFERENCES
  };
}

type ProviderProps = { initial: ProfileState | undefined; onFinish: () => void; children: React.ReactNode };

export function OnboardingProvider({ initial, onFinish, children }: ProviderProps) {
  const [draft, setDraft] = useState(() => draftFromProfile(initial));
  const value = useMemo<OnboardingContextValue>(
    () => ({
      draft,
      update: (patch) => setDraft((current) => ({ ...current, ...patch })),
      finish: onFinish
    }),
    [draft, onFinish]
  );
  return <OnboardingContext.Provider value={value}>{children}</OnboardingContext.Provider>;
}

export function useOnboarding(): OnboardingContextValue {
  const context = useContext(OnboardingContext);
  if (!context) {
    throw new Error("useOnboarding must be used within OnboardingProvider");
  }
  return context;
}
