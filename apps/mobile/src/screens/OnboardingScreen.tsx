import React, { useState } from "react";
import { View } from "react-native";
import { DEFAULT_PREFERENCES, type Preferences } from "../api/profile";
import { useAuth } from "../hooks/useAuth";
import { useMyProfile, usePreferences, useSavePreferences, useSaveProfile } from "../hooks/useProfileData";
import { messageFromError } from "../lib/errors";
import { formatCounter } from "../lib/format";
import { firstMissingStep, ONBOARDING_STEPS, progressFraction } from "../lib/onboarding";
import { profileToFormValues, profileToPrivacy, toProfileInput } from "../lib/profileForm";
import { strings } from "../lib/strings";
import {
  hasErrors,
  validatePreferences,
  validateProfileForm,
  type PreferencesErrors,
  type PreferencesValues,
  type ProfileFormErrors,
  type ProfileFormValues
} from "../lib/validation";
import { ErrorView } from "../shared/feedback/ErrorView";
import { LoadingView } from "../shared/feedback/LoadingView";
import { AboutFields, BasicsFields } from "../shared/forms/ProfileFields";
import { LocationCard } from "../shared/forms/LocationCard";
import { PhotoManager } from "../shared/forms/PhotoManager";
import { PreferencesFields } from "../shared/forms/PreferencesFields";
import { KeyboardScreen } from "../shared/layout/KeyboardScreen";
import { SafeAreaLayout } from "../shared/layout/SafeAreaLayout";
import { Button } from "../shared/ui/Button";
import { Notice } from "../shared/ui/Notice";
import { ProgressBar } from "../shared/ui/ProgressBar";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";

const makeStyles = (t: Theme) => ({
  head: { gap: t.spacing.xs },
  brand: { color: t.colors.primary, letterSpacing: 4 },
  nav: { flexDirection: "row" as const, gap: t.spacing.sm },
  navGrow: { flex: 1 },
  top: { flexDirection: "row" as const, justifyContent: "space-between" as const, alignItems: "center" as const }
});

/** Shown while `profileComplete` is false. Loads server state first, then resumes at the first missing step. */
export default function OnboardingScreen() {
  const profileQuery = useMyProfile();
  const preferencesQuery = usePreferences();

  if (profileQuery.isPending || preferencesQuery.isPending) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Loading your profile..." />
      </SafeAreaLayout>
    );
  }
  if (
    (profileQuery.isError && profileQuery.data === undefined) ||
    (preferencesQuery.isError && preferencesQuery.data === undefined)
  ) {
    return (
      <SafeAreaLayout>
        <ErrorView
          message={messageFromError(profileQuery.error ?? preferencesQuery.error)}
          onAction={() => {
            void profileQuery.refetch();
            void preferencesQuery.refetch();
          }}
        />
      </SafeAreaLayout>
    );
  }

  return (
    <Wizard
      initialPreferences={preferencesQuery.data ?? DEFAULT_PREFERENCES}
      initialStep={firstMissingStep({
        hasProfile: profileQuery.data != null,
        hasLocation: profileQuery.data?.hasLocation ?? false,
        photoCount: profileQuery.data?.photos.length ?? 0
      })}
    />
  );
}

type WizardProps = { initialStep: number; initialPreferences: Preferences };

function Wizard({ initialPreferences, initialStep }: WizardProps) {
  const styles = useThemedStyles(makeStyles);
  const { refreshProfileState, signOut } = useAuth();
  const profileQuery = useMyProfile();
  const profile = profileQuery.data ?? null;
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();

  const [step, setStep] = useState(initialStep);
  const [form, setForm] = useState<ProfileFormValues>(() => profileToFormValues(profile));
  const [formErrors, setFormErrors] = useState<ProfileFormErrors>({});
  const [prefs, setPrefs] = useState<PreferencesValues>(() => ({ ...initialPreferences }));
  const [prefErrors, setPrefErrors] = useState<PreferencesErrors>({});
  const [locationSaved, setLocationSaved] = useState(profile?.hasLocation ?? false);
  const [error, setError] = useState<string | null>(null);
  const [finishing, setFinishing] = useState(false);

  const stepKey = ONBOARDING_STEPS[step] ?? "basics";
  const saving = saveProfile.isPending || savePreferences.isPending;
  const photos = profile?.photos ?? [];

  function goTo(next: number) {
    setError(null);
    setStep(Math.min(Math.max(next, 0), ONBOARDING_STEPS.length - 1));
  }

  async function persistProfile(values: ProfileFormValues): Promise<boolean> {
    const errors = validateProfileForm(values);
    setFormErrors(errors);
    if (hasErrors(errors)) {
      return false;
    }
    try {
      await saveProfile.mutateAsync(toProfileInput(values, profileToPrivacy(profile)));
      return true;
    } catch (err) {
      setError(messageFromError(err));
      return false;
    }
  }

  async function next() {
    setError(null);
    if (stepKey === "basics") {
      if (await persistProfile(form)) {
        goTo(step + 1);
      }
    } else if (stepKey === "preferences") {
      const errors = validatePreferences(prefs);
      setPrefErrors(errors);
      if (hasErrors(errors)) {
        return;
      }
      try {
        await savePreferences.mutateAsync(prefs);
        goTo(step + 1);
      } catch (err) {
        setError(messageFromError(err));
      }
    } else if (stepKey === "about") {
      if (await persistProfile(form)) {
        goTo(step + 1);
      }
    } else {
      goTo(step + 1);
    }
  }

  async function finish() {
    setError(null);
    setFinishing(true);
    try {
      const me = await refreshProfileState();
      if (me && !me.profileComplete) {
        setError("Almost there. Make sure you have shared your location and added at least one photo.");
      }
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setFinishing(false);
    }
  }

  const nextDisabled = stepKey === "location" && !locationSaved;

  return (
    <KeyboardScreen testID="onboarding-screen">
      <View style={styles.top}>
        <Text style={styles.brand} variant="eyebrow" weight="bold">
          {strings.appName}
        </Text>
        <Button label="Sign out" onPress={() => void signOut()} size="sm" testID="onboarding-signout" variant="ghost" />
      </View>
      <View style={styles.head}>
        <ProgressBar label="Profile setup progress" value={progressFraction(step)} />
        <Text tone="muted" variant="caption">
          Step {step + 1} of {ONBOARDING_STEPS.length}: {strings.onboarding.steps[step]}
        </Text>
        <Text accessibilityRole="header" variant="title" weight="bold">
          {stepTitle(stepKey)}
        </Text>
      </View>

      {stepKey === "basics" ? <BasicsFields errors={formErrors} onChange={setForm} values={form} /> : null}
      {stepKey === "preferences" ? (
        <PreferencesFields errors={prefErrors} onChange={setPrefs} values={prefs} />
      ) : null}
      {stepKey === "about" ? (
        <>
          <AboutFields errors={formErrors} onChange={setForm} values={form} />
          <Text tone="muted" variant="caption">
            Bio {formatCounter(form.bio.length, 500)}. You can skip this now and finish later from your profile.
          </Text>
        </>
      ) : null}
      {stepKey === "location" ? (
        <LocationCard hasLocation={locationSaved} onSaved={() => setLocationSaved(true)} />
      ) : null}
      {stepKey === "photos" ? (
        <>
          <Text tone="muted">{strings.onboarding.photosHint}</Text>
          <PhotoManager photos={photos} />
        </>
      ) : null}

      {error ? <Notice testID="onboarding-error" title={error} tone="danger" /> : null}

      <View style={styles.nav}>
        {step > 0 ? (
          <Button disabled={saving || finishing} label="Back" onPress={() => goTo(step - 1)} testID="onboarding-back" variant="outline" />
        ) : null}
        <View style={styles.navGrow}>
          {stepKey === "photos" ? (
            <Button
              disabled={photos.length < 1}
              label="Finish"
              loading={finishing}
              onPress={() => void finish()}
              size="lg"
              testID="onboarding-finish"
            />
          ) : (
            <Button
              disabled={nextDisabled}
              label="Continue"
              loading={saving}
              onPress={() => void next()}
              size="lg"
              testID="onboarding-next"
            />
          )}
        </View>
      </View>
      {stepKey === "about" ? (
        <Button label="Skip for now" onPress={() => goTo(step + 1)} testID="onboarding-skip" variant="ghost" />
      ) : null}
    </KeyboardScreen>
  );
}

function stepTitle(key: (typeof ONBOARDING_STEPS)[number]): string {
  switch (key) {
    case "basics":
      return "Let's start with you";
    case "preferences":
      return "Who would you like to meet?";
    case "about":
      return "Tell your story";
    case "location":
      return "Find people near you";
    case "photos":
      return "Add your best photos";
  }
}
