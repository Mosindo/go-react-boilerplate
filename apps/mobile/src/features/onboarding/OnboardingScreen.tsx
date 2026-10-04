import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { getErrorMessage } from "../../api/client";
import * as profileApi from "../../api/profile";
import { queryKeys } from "../../api/queryKeys";
import type { OwnProfile } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { useOwnProfile, useSaveProfile, useSavePreferences } from "../../hooks/useProfile";
import { maskBirthInput } from "../../lib/dates";
import { BIO_MAX } from "../../lib/validation";
import { setOnboardingHold } from "../../store/onboardingHold";
import { radii, spacing, useColors } from "../../theme";
import { Button, Input, Screen, Text, toast } from "../../ui";
import { GenderPicker } from "../profile/GenderPicker";
import { InterestPicker } from "../profile/InterestPicker";
import { LocationField } from "../profile/LocationField";
import { PhotoManager } from "../profile/PhotoManager";
import { PreferencesFields } from "../profile/PreferencesFields";
import {
  STEPS,
  initialWizardState,
  toProfileInput,
  validateStep,
  type Step,
  type StepErrors,
  type WizardState
} from "./wizard";

const TITLES: Record<Step, { title: string; subtitle: string }> = {
  about: { title: "Let's start with you", subtitle: "Just the basics. You can change most of this later." },
  preferences: { title: "Who would you like to meet?", subtitle: "Only people matching this will show up." },
  location: { title: "Where are you?", subtitle: "Helps you meet people nearby." },
  story: { title: "A little about you", subtitle: "A short bio and a few interests make a first message easier." },
  photos: { title: "Add your photos", subtitle: "Add at least one. Your first photo is your main one." }
};

export function OnboardingScreen() {
  const colors = useColors();
  const client = useQueryClient();
  const { logout } = useAuth();
  const profileQuery = useOwnProfile();
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();

  const [stepIndex, setStepIndex] = useState(0);
  const [state, setState] = useState<WizardState>(() => initialWizardState(profileQuery.data));
  const [errors, setErrors] = useState<StepErrors>({});
  const [busy, setBusy] = useState(false);
  const step = STEPS[stepIndex];
  const profile: OwnProfile | null | undefined = profileQuery.data;

  useEffect(() => {
    setOnboardingHold(true);
    return () => setOnboardingHold(false);
  }, []);

  const patch = (partial: Partial<WizardState>) => setState((current) => ({ ...current, ...partial }));

  async function persistProfile(): Promise<boolean> {
    setBusy(true);
    try {
      const saved = await saveProfile.mutateAsync(toProfileInput(state, profile));
      void saved;
      await savePreferences.mutateAsync(state.preferences);
      if (state.coordinates) {
        await profileApi.setLocation(state.coordinates.latitude, state.coordinates.longitude);
        client.setQueryData<OwnProfile | null>(queryKeys.profile, (p) => (p ? { ...p, hasLocation: true } : p));
      }
      return true;
    } catch (error) {
      toast.error(getErrorMessage(error));
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function next() {
    const found = validateStep(step, state);
    setErrors(found);
    if (Object.keys(found).length > 0) return;
    if (step === "story") {
      if (!(await persistProfile())) return;
    }
    setStepIndex((index) => Math.min(STEPS.length - 1, index + 1));
  }

  async function finish() {
    setBusy(true);
    try {
      const fresh = await client.fetchQuery({
        queryKey: queryKeys.profile,
        queryFn: profileApi.getOwnProfile,
        staleTime: 0
      });
      if (!fresh || fresh.photos.length === 0) {
        toast.error("Add at least one photo to finish.");
        return;
      }
      client.setQueryData(queryKeys.profile, { ...fresh, completed: true });
      setOnboardingHold(false); // the gate re-evaluates and shows the app
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setBusy(false);
    }
  }

  const photoCount = profile?.photos.length ?? 0;
  const { title, subtitle } = TITLES[step];

  return (
    <Screen edges={["top", "left", "right", "bottom"]} keyboardAvoiding scroll>
      <View style={styles.progressRow}>
        {STEPS.map((name, index) => (
          <View
            key={name}
            style={[styles.progress, { backgroundColor: index <= stepIndex ? colors.primary : colors.border }]}
          />
        ))}
      </View>
      <Text accessibilityLabel={`Step ${stepIndex + 1} of ${STEPS.length}`} tone="muted" variant="small">
        Step {stepIndex + 1} of {STEPS.length}
      </Text>
      <Text variant="heading">{title}</Text>
      <Text tone="muted">{subtitle}</Text>

      <View style={styles.body}>
        {step === "about" ? (
          <>
            <Input
              autoCapitalize="words"
              error={errors.displayName}
              label="First name"
              maxLength={40}
              onChangeText={(displayName) => patch({ displayName })}
              textContentType="givenName"
              value={state.displayName}
            />
            <Input
              error={errors.birthInput}
              hint="Lumen is for adults. You must be 18 or older."
              keyboardType="number-pad"
              label="Birth date (DD/MM/YYYY)"
              maxLength={10}
              onChangeText={(text) => patch({ birthInput: maskBirthInput(text) })}
              placeholder="DD/MM/YYYY"
              value={state.birthInput}
            />
            <GenderPicker error={errors.gender} onChange={(gender) => patch({ gender })} value={state.gender} />
          </>
        ) : null}

        {step === "preferences" ? (
          <>
            <PreferencesFields onChange={(preferences) => patch({ preferences })} value={state.preferences} />
            {errors.preferences ? (
              <Text accessibilityRole="alert" tone="danger">
                {errors.preferences}
              </Text>
            ) : null}
          </>
        ) : null}

        {step === "location" ? (
          <LocationField
            city={state.city}
            coordinates={state.coordinates}
            error={errors.city}
            hasSavedLocation={profile?.hasLocation}
            onCityChange={(city) => patch({ city })}
            onCoordinates={(coordinates) => patch({ coordinates })}
          />
        ) : null}

        {step === "story" ? (
          <>
            <Input
              error={errors.bio}
              hint={`${state.bio.length}/${BIO_MAX}`}
              label="Bio (optional)"
              maxLength={BIO_MAX}
              multiline
              onChangeText={(bio) => patch({ bio })}
              placeholder="What makes a good Sunday for you?"
              value={state.bio}
            />
            <Text weight="semibold">Interests</Text>
            <InterestPicker onChange={(interests) => patch({ interests })} selected={state.interests} />
          </>
        ) : null}

        {step === "photos" ? <PhotoManager name={state.displayName} photos={profile?.photos ?? []} /> : null}
      </View>

      <View style={styles.footer}>
        {step === "photos" ? (
          <Button
            disabled={photoCount === 0}
            label={photoCount === 0 ? "Add a photo to finish" : "Finish and start meeting people"}
            loading={busy}
            onPress={finish}
          />
        ) : (
          <Button label={step === "story" ? "Save and continue" : "Continue"} loading={busy} onPress={next} />
        )}
        {stepIndex > 0 ? (
          <Button
            disabled={busy}
            label="Back"
            onPress={() => setStepIndex((i) => Math.max(0, i - 1))}
            variant="ghost"
          />
        ) : (
          <Button label="Sign out" onPress={() => void logout()} variant="ghost" />
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progressRow: { flexDirection: "row", gap: spacing.xxs, paddingTop: spacing.sm },
  progress: { flex: 1, height: 4, borderRadius: radii.pill },
  body: { gap: spacing.md, marginTop: spacing.xs },
  footer: { gap: spacing.xs, marginTop: spacing.md }
});
