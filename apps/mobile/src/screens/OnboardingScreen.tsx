import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage, listInterests, queryKeys, savePreferences, saveProfile, type OwnProfile, type Preferences } from "../api/platform";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesFields, defaultPreferences } from "../components/PreferencesFields";
import { ProfileFields, emptyDraft, validateDraft, type DraftErrors, type ProfileDraft } from "../components/ProfileFields";
import { Screen } from "../components/Screen";
import { useDeviceLocation } from "../components/useDeviceLocation";
import { useAuth } from "../hooks/useAuth";
import { Button, Notice, Text, colors, radii, spacing } from "../shared/ui";

type Step = 0 | 1 | 2 | 3;
const STEP_COPY = [
  { title: "About you", subtitle: "The basics people see first." },
  { title: "Who you'd like to meet", subtitle: "You can change this any time." },
  { title: "Your story", subtitle: "A bio, your interests and where you are." },
  { title: "Add your photos", subtitle: "At least one photo is needed to start meeting people." }
] as const;

type Props = { profile: OwnProfile | null; onFinish: () => void };

/** First-run wizard. Photos come last because the server needs the profile to exist first. */
export default function OnboardingScreen({ onFinish, profile }: Props) {
  const client = useQueryClient();
  const { logout } = useAuth();
  const [step, setStep] = useState<Step>(profile ? 3 : 0);
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [errors, setErrors] = useState<DraftErrors>({});
  const [prefs, setPrefs] = useState<Preferences>(defaultPreferences);
  const [error, setError] = useState<string | null>(null);
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const location = useDeviceLocation();
  const interests = useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: Infinity });

  const create = useMutation({
    mutationFn: async () => {
      const { input } = validateDraft(draft);
      if (!input) {
        throw new Error("Please complete your details.");
      }
      await saveProfile({ ...input, ...(coords ?? {}) });
      await savePreferences(prefs);
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: queryKeys.profile });
      setStep(3);
    },
    onError: (e) => setError(errorMessage(e))
  });

  const locateMe = async () => {
    const found = await location.request();
    if (found) {
      setCoords({ latitude: found.latitude, longitude: found.longitude });
      if (!draft.city && found.city) {
        setDraft({ ...draft, city: found.city });
      }
    }
  };

  const next = () => {
    setError(null);
    if (step === 0) {
      const result = validateDraft(draft);
      const basics: DraftErrors = { firstName: result.errors.firstName, birthDate: result.errors.birthDate, gender: result.errors.gender };
      setErrors(basics);
      if (basics.firstName || basics.birthDate || basics.gender) {
        return;
      }
      setStep(1);
    } else if (step === 1) {
      setStep(2);
    } else if (step === 2) {
      const result = validateDraft(draft);
      setErrors(result.errors);
      if (!result.input) {
        setStep(0);
        return;
      }
      create.mutate();
    }
  };

  const hasPhoto = (profile?.photos.length ?? 0) > 0;
  const copy = STEP_COPY[step];

  return (
    <Screen edges={["top", "right", "bottom", "left"]} testID="onboarding-screen">
      <View style={styles.progress} accessibilityLabel={`Step ${step + 1} of 4`}>
        {STEP_COPY.map((_, i) => (
          <View key={i} style={[styles.bar, i <= step ? styles.barActive : null]} />
        ))}
      </View>
      <View style={styles.copy}>
        <Text variant="title" weight="bold">
          {copy.title}
        </Text>
        <Text tone="muted">{copy.subtitle}</Text>
      </View>
      {error ? <Notice description={error} title="We could not save that" tone="danger" /> : null}

      {step === 0 ? <ProfileFields draft={draft} errors={errors} interests={[]} onChange={setDraft} section="basics" /> : null}
      {step === 1 ? <PreferencesFields onChange={setPrefs} value={prefs} /> : null}
      {step === 2 ? (
        <>
          <ProfileFields draft={draft} errors={errors} interests={interests.data ?? []} onChange={setDraft} section="story" />
          <View style={styles.location}>
            <Text weight="semibold">Find people near you</Text>
            <Text tone="muted" variant="caption">
              We store your position rounded to about 1 km and only ever show an approximate distance.
            </Text>
            <Button
              label={coords ? "Location saved ✓" : "Use my location"}
              loading={location.status === "loading"}
              onPress={() => void locateMe()}
              size="sm"
              testID="onboarding-location"
              variant={coords ? "success" : "secondary"}
            />
            {location.status === "denied" ? (
              <Text tone="muted" variant="caption">
                Location is off. You can still use the app; add it later in your profile to see people nearby.
              </Text>
            ) : null}
          </View>
        </>
      ) : null}
      {step === 3 ? <PhotoManager /> : null}

      <View style={styles.actions}>
        {step < 3 ? (
          <Button label={step === 2 ? "Save and continue" : "Continue"} loading={create.isPending} onPress={next} size="lg" testID="onboarding-next" />
        ) : (
          <Button disabled={!hasPhoto} label="Start meeting people" onPress={onFinish} size="lg" testID="onboarding-finish" />
        )}
        {step > 0 && step < 3 ? <Button label="Back" onPress={() => setStep((s) => (s - 1) as Step)} variant="ghost" /> : null}
        {step === 0 ? <Button label="Sign out" onPress={() => void logout()} variant="ghost" /> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: radii.pill, backgroundColor: colors.surfaceSubtle },
  barActive: { backgroundColor: colors.primary },
  copy: { gap: spacing.xs },
  location: { gap: spacing.sm, padding: spacing.lg, borderRadius: radii.lg, backgroundColor: colors.surfaceAccent, borderWidth: 1, borderColor: colors.primaryBorder },
  actions: { gap: spacing.sm, marginTop: spacing.md }
});
