import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/profile";
import type { MeProfile, Preferences } from "../../api/types";
import { PhotoGrid } from "../../components/PhotoGrid";
import { PreferencesFields } from "../../components/PreferencesFields";
import { ProfileFields } from "../../components/ProfileFields";
import {
  defaultPreferences,
  formToInput,
  profileToForm,
  validatePreferences,
  type ProfileFormErrors,
  type ProfileFormState
} from "../../components/profileForm";
import { shareLocation } from "../../components/location";
import { useSetMyProfile } from "../../hooks/queries";
import { useSession } from "../../hooks/useSession";
import { colors, radii, spacing } from "../../theme";
import { Button, Header, IconButton, Screen, Text } from "../../ui";

type Step = "basics" | "preferences" | "about" | "photos" | "location";
const steps: Step[] = ["basics", "preferences", "about", "photos", "location"];

const titles: Record<Step, { title: string; subtitle: string }> = {
  basics: { title: "Faisons connaissance", subtitle: "Ces informations de base composent votre profil." },
  preferences: { title: "Qui aimeriez-vous rencontrer ?", subtitle: "Vous pourrez changer ces réglages à tout moment." },
  about: { title: "Parlez un peu de vous", subtitle: "Une bio et quelques centres d'intérêt aident à briser la glace." },
  photos: { title: "Ajoutez vos photos", subtitle: "Au moins une photo, où l'on voit bien votre visage." },
  location: { title: "Où êtes-vous ?", subtitle: "Pour vous proposer des personnes proches de vous." }
};

function initialStep(me: MeProfile): number {
  const missing = me.completeness.missing;
  if (missing.includes("profile")) return 0;
  if (missing.includes("preferences")) return 1;
  if (missing.includes("photos")) return 3;
  if (missing.includes("location")) return 4;
  return 0;
}

export function OnboardingScreen({ me }: { me: MeProfile }) {
  const { signOut } = useSession();
  const setMyProfile = useSetMyProfile();
  const [index, setIndex] = useState(() => initialStep(me));
  const [form, setForm] = useState<ProfileFormState>(() => profileToForm(me.profile));
  const [errors, setErrors] = useState<ProfileFormErrors>({});
  const [prefs, setPrefs] = useState<Preferences>(() => (me.preferences.interestedIn.length ? me.preferences : defaultPreferences));
  const [prefsError, setPrefsError] = useState<string | null>(null);
  const [locationIssue, setLocationIssue] = useState<string | null>(null);
  const step = steps[index];

  const save = useMutation({
    mutationFn: async (): Promise<MeProfile | null> => {
      switch (step) {
        case "basics":
        case "about": {
          const { input, errors: next } = formToInput(form);
          setErrors(next);
          return input ? profileApi.update(input) : null;
        }
        case "preferences": {
          const problem = validatePreferences(prefs);
          setPrefsError(problem);
          return problem ? null : profileApi.updatePreferences(prefs);
        }
        case "photos":
          return me.photos.length > 0 ? me : null;
        case "location": {
          const result = await shareLocation(true);
          if (!result.ok) {
            setLocationIssue(
              result.reason === "denied"
                ? "L'accès à la position a été refusé. Autorisez-le dans les réglages de votre appareil pour continuer."
                : "Position indisponible pour le moment. Réessayez dans un endroit dégagé."
            );
            return null;
          }
          return result.profile;
        }
      }
    },
    onSuccess: (profile) => {
      if (!profile) {
        return;
      }
      setMyProfile(profile);
      if (index < steps.length - 1) {
        setIndex(index + 1);
      }
    }
  });

  const progress = useMemo(() => (index + 1) / steps.length, [index]);
  const canContinue = step !== "photos" || me.photos.length > 0;

  return (
    <Screen
      scroll
      testID={`onboarding-${step}`}
      footer={
        <>
          {save.error ? (
            <Text variant="small" color="danger" align="center">
              {errorMessage(save.error)}
            </Text>
          ) : null}
          <Button
            label={step === "location" ? "Partager ma position approximative" : "Continuer"}
            icon={step === "location" ? "navigate-outline" : undefined}
            onPress={() => save.mutate()}
            loading={save.isPending}
            disabled={!canContinue}
            testID="onboarding-continue"
          />
        </>
      }
    >
      <Header
        onBack={index > 0 ? () => setIndex(index - 1) : undefined}
        right={<IconButton icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} color={colors.textSubtle} />}
      />
      <View style={styles.progressTrack} accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: steps.length, now: index + 1 }}>
        <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
      </View>
      <View style={styles.titles}>
        <Text variant="caption" color="primary">
          Étape {index + 1} sur {steps.length}
        </Text>
        <Text variant="title">{titles[step].title}</Text>
        <Text variant="small" color="textMuted">
          {titles[step].subtitle}
        </Text>
      </View>

      {step === "basics" ? (
        <ProfileFields form={form} errors={errors} onChange={(p) => setForm({ ...form, ...p })} sections={["basics"]} />
      ) : null}
      {step === "preferences" ? <PreferencesFields value={prefs} onChange={setPrefs} error={prefsError} /> : null}
      {step === "about" ? (
        <ProfileFields form={form} errors={errors} onChange={(p) => setForm({ ...form, ...p })} sections={["about"]} />
      ) : null}
      {step === "photos" ? <PhotoGrid photos={me.photos} /> : null}
      {step === "location" ? (
        <View style={styles.locationCard}>
          <Ionicons name="location-outline" size={28} color={colors.gold} />
          <Text variant="body">Nous utilisons uniquement une position approximative (arrondie à environ 1 km).</Text>
          <Text variant="small" color="textMuted">
            Les autres membres voient seulement une distance arrondie, jamais votre adresse ni vos coordonnées. Vous pouvez masquer la
            distance dans Confidentialité.
          </Text>
          {locationIssue ? (
            <Text variant="small" color="danger" testID="location-error">
              {locationIssue}
            </Text>
          ) : null}
        </View>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  progressTrack: {
    height: 4,
    borderRadius: 2,
    backgroundColor: colors.surfaceHigh,
    overflow: "hidden"
  },
  progressFill: {
    height: 4,
    backgroundColor: colors.primary
  },
  titles: {
    gap: spacing.xs,
    paddingTop: spacing.lg,
    paddingBottom: spacing.lg
  },
  locationCard: {
    gap: spacing.sm,
    padding: spacing.lg,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  }
});
