import React, { useEffect, useMemo, useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { friendlyError } from "../api/errors";
import { savePreferences, saveProfile } from "../api/platform";
import type { MeProfile, Preferences } from "../api/types";
import { MAX_PHOTOS, PhotoManager } from "../components/PhotoManager";
import {
  AboutFields,
  IdentityFields,
  LocationPicker,
  PreferencesFields,
  type AboutState,
  type IdentityState,
  type LocationState
} from "../components/profileFields";
import { queryKeys, useInterests } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { isoToParts, toIsoDate, validateBirthDate, validateFirstName } from "../lib/validation";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Notice, Text, colors, radii, spacing } from "../shared/ui";

const STEPS = ["Qui êtes-vous ?", "Qui cherchez-vous ?", "Où êtes-vous ?", "Vos photos", "Votre histoire"] as const;

function initialIdentity(me: MeProfile): IdentityState {
  return {
    firstName: me.profile?.firstName ?? "",
    birthDate: me.profile ? isoToParts(me.profile.birthDate) : { day: "", month: "", year: "" },
    gender: me.profile?.gender ?? null
  };
}

/** First-run wizard. The account only becomes visible to others once all steps are done. */
export default function OnboardingScreen({ me }: { me: MeProfile }) {
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const interests = useInterests();

  const [step, setStep] = useState(0);
  const [identity, setIdentity] = useState<IdentityState>(() => initialIdentity(me));
  const [preferences, setPreferences] = useState<Preferences>(
    me.preferences ?? { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 }
  );
  const [location, setLocation] = useState<LocationState>(
    me.profile ? { latitude: me.profile.latitude, longitude: me.profile.longitude, city: me.profile.city } : null
  );
  const [about, setAbout] = useState<AboutState>({
    bio: me.profile?.bio ?? "",
    occupation: me.profile?.occupation ?? "",
    lookingFor: me.profile?.lookingFor ?? "open",
    interests: me.profile?.interests ?? []
  });
  const [errors, setErrors] = useState<{ firstName?: string | null; birthDate?: string | null; gender?: string | null }>({});
  const [stepError, setStepError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // A step error goes away once the user has acted on it (e.g. added the missing photo).
  const photoCount = me.photos.length;
  useEffect(() => {
    setStepError(null);
  }, [photoCount]);

  const locked = Boolean(me.profile); // an existing profile fixes the date of birth

  const validators = useMemo(
    () => [
      () => {
        const next = {
          firstName: validateFirstName(identity.firstName),
          birthDate: locked ? null : validateBirthDate(identity.birthDate),
          gender: identity.gender ? null : "Choisissez une option."
        };
        setErrors(next);
        return !next.firstName && !next.birthDate && !next.gender;
      },
      () => {
        if (preferences.interestedIn.length === 0) {
          setStepError("Choisissez au moins une option.");
          return false;
        }
        return true;
      },
      () => {
        if (!location) {
          setStepError("Indiquez votre position ou votre ville pour trouver des profils proches.");
          return false;
        }
        return true;
      },
      () => {
        if (me.photos.length === 0) {
          setStepError("Ajoutez au moins une photo.");
          return false;
        }
        return true;
      },
      () => true
    ],
    [identity, locked, location, me.photos.length, preferences.interestedIn.length]
  );

  const next = () => {
    setStepError(null);
    if (validators[step]()) {
      setStep((current) => Math.min(current + 1, STEPS.length - 1));
    }
  };

  const finish = async () => {
    const birthIso = me.profile?.birthDate ?? toIsoDate(identity.birthDate);
    if (!birthIso || !identity.gender || !location) {
      setStepError("Des informations obligatoires sont manquantes. Revenez en arrière pour les compléter.");
      return;
    }
    setSaving(true);
    setStepError(null);
    try {
      await saveProfile({
        firstName: identity.firstName.trim(),
        birthDate: birthIso,
        gender: identity.gender,
        bio: about.bio.trim(),
        occupation: about.occupation.trim(),
        lookingFor: about.lookingFor,
        city: location.city,
        latitude: location.latitude,
        longitude: location.longitude,
        interests: about.interests
      });
      await savePreferences(preferences);
      await queryClient.invalidateQueries({ queryKey: queryKeys.me });
    } catch (error) {
      setStepError(friendlyError(error));
    } finally {
      setSaving(false);
    }
  };

  const last = step === STEPS.length - 1;

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled" testID="screen-onboarding">
          <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
            {STEPS.map((title, index) => (
              <View key={title} style={[styles.bar, index <= step ? styles.barDone : null]} />
            ))}
          </View>
          <Text tone="primary" variant="eyebrow" weight="bold">
            Étape {step + 1} sur {STEPS.length}
          </Text>
          <Text variant="title" weight="bold">
            {STEPS[step]}
          </Text>

          {step === 0 ? <IdentityFields birthDateLocked={locked} errors={errors} onChange={setIdentity} value={identity} /> : null}
          {step === 1 ? <PreferencesFields onChange={setPreferences} value={preferences} /> : null}
          {step === 2 ? <LocationPicker onChange={setLocation} value={location} /> : null}
          {step === 3 ? (
            <View style={styles.stack}>
              <Text tone="muted">Ajoutez jusqu'à {MAX_PHOTOS} photos. Montrez votre visage et ce que vous aimez faire.</Text>
              <PhotoManager photos={me.photos} />
            </View>
          ) : null}
          {step === 4 ? (
            interests.data ? (
              <AboutFields catalogue={interests.data} onChange={setAbout} value={about} />
            ) : (
              <Text tone="muted">Chargement des centres d'intérêt…</Text>
            )
          ) : null}

          {stepError ? <Notice title={stepError} tone="danger" /> : null}

          <View style={styles.buttons}>
            {step > 0 ? (
              <Button
                disabled={saving}
                label="Retour"
                onPress={() => {
                  setStepError(null);
                  setStep(step - 1);
                }}
                style={styles.flex}
                variant="outline"
              />
            ) : null}
            {last ? (
              <Button label="Terminer" loading={saving} onPress={() => void finish()} style={styles.flex} testID="onboarding-finish" />
            ) : (
              <Button label="Continuer" onPress={next} style={styles.flex} testID="onboarding-next" />
            )}
          </View>
          <Button label="Se déconnecter" onPress={() => void signOut()} size="sm" variant="ghost" />
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { padding: spacing.xl, gap: spacing.lg, paddingBottom: spacing.xxxl },
  stack: { gap: spacing.lg },
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: radii.pill, backgroundColor: colors.surfaceSubtle },
  barDone: { backgroundColor: colors.primary },
  buttons: { flexDirection: "row", gap: spacing.md, marginTop: spacing.md }
});
