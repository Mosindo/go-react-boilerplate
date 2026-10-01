import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { profileApi } from "../api/platform";
import { GENDERS, type Gender, type Preferences } from "../api/types";
import { InterestPicker } from "../components/InterestPicker";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesEditor } from "../components/PreferencesEditor";
import { useAuth } from "../hooks/useAuth";
import { useDeviceLocation } from "../hooks/useDeviceLocation";
import { ErrorView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text, spacing, useTheme } from "../shared/ui";
import { formatBirthDateInput, parseBirthDate } from "../utils/validation";

const STEPS = ["Vous", "Vos envies", "Photos", "À propos", "Localisation"] as const;
const DEFAULT_PREFS: Preferences = { interestedIn: ["woman", "man", "non_binary", "other"], ageMin: 18, ageMax: 60, maxDistanceKm: 50 };

/**
 * Five short steps. Each step saves on "Continuer", so leaving the app halfway
 * resumes where the person stopped (the first step is chosen from server state).
 */
export default function OnboardingScreen() {
  const { colors } = useTheme();
  const { user, refreshUser, signOut } = useAuth();
  const existing = useQuery({
    queryKey: ["onboarding", "profile"],
    queryFn: async () => {
      try {
        return await profileApi.getOwn();
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) {
          return null; // no profile yet: start from the first step
        }
        throw e;
      }
    }
  });

  const [step, setStep] = useState(0);
  const [initialized, setInitialized] = useState(false);
  const [firstName, setFirstName] = useState("");
  const [birth, setBirth] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [prefs, setPrefs] = useState<Preferences>(DEFAULT_PREFS);
  const [bio, setBio] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [city, setCity] = useState("");
  const [photoCount, setPhotoCount] = useState(user?.photoCount ?? 0);
  const [error, setError] = useState<string | null>(null);
  const location = useDeviceLocation((found) => found && setCity(found));

  if (existing.isSuccess && !initialized) {
    setInitialized(true);
    const profile = existing.data;
    if (profile) {
      setFirstName(profile.firstName);
      setGender(profile.gender);
      setBio(profile.bio);
      setCity(profile.city);
      setInterests(profile.interests.map((i) => i.slug));
      setBirth(profile.birthDate.split("-").reverse().join("/"));
      setPhotoCount(profile.photos.length);
      setStep(profile.photos.length === 0 ? 2 : 3);
    }
  }

  const fail = (e: unknown) => setError(e instanceof ApiError ? e.message : "Une erreur est survenue.");
  const hasProfile = existing.data !== null && existing.data !== undefined;

  const saveBasics = useMutation({
    mutationFn: async () => {
      const parsed = parseBirthDate(birth);
      if (!parsed.ok || !gender) {
        throw new ApiError(400, "Vérifiez vos informations.");
      }
      await profileApi.save({ firstName: firstName.trim(), birthDate: parsed.iso, gender, bio, city });
      await profileApi.savePreferences(prefs);
    },
    onSuccess: () => {
      setError(null);
      setStep(2);
    },
    onError: fail
  });

  const saveAbout = useMutation({
    mutationFn: async () => {
      if (!gender) {
        return;
      }
      await profileApi.save({ firstName: firstName.trim(), gender, bio, city, interests });
    },
    onSuccess: () => {
      setError(null);
      setStep(4);
    },
    onError: fail
  });

  const finish = useMutation({
    mutationFn: async () => {
      if (gender) {
        await profileApi.save({ firstName: firstName.trim(), gender, bio, city, interests });
      }
      return refreshUser();
    },
    onError: fail
  });

  const next = () => {
    setError(null);
    if (step === 0) {
      const parsed = parseBirthDate(birth);
      if (firstName.trim().length === 0) {
        setError("Indiquez votre prénom.");
      } else if (!parsed.ok) {
        setError(parsed.error);
      } else if (!gender) {
        setError("Choisissez une option pour votre genre.");
      } else {
        setStep(1);
      }
    } else if (step === 1) {
      saveBasics.mutate();
    } else if (step === 2) {
      if (photoCount === 0) {
        setError("Ajoutez au moins une photo pour continuer.");
      } else {
        setStep(3);
      }
    } else if (step === 3) {
      saveAbout.mutate();
    } else {
      finish.mutate();
    }
  };

  const busy = saveBasics.isPending || saveAbout.isPending || finish.isPending;

  if (existing.isError) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger votre profil." onAction={() => void existing.refetch()} />
      </SafeAreaLayout>
    );
  }

  if (existing.isLoading) {
    return (
      <SafeAreaLayout>
        <View style={styles.center}>
          <Text tone="muted">Chargement…</Text>
        </View>
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <View accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length} : ${STEPS[step]}`} accessible style={styles.progress}>
            {STEPS.map((label, i) => (
              <View key={label} style={[styles.bar, { backgroundColor: i <= step ? colors.primary : colors.border }]} />
            ))}
          </View>

          {step === 0 ? (
            <View style={styles.form}>
              <Text accessibilityRole="header" variant="title">
                Faisons connaissance
              </Text>
              <FormField label="Prénom">
                <Input accessibilityLabel="Prénom" autoCapitalize="words" maxLength={50} onChangeText={setFirstName} value={firstName} />
              </FormField>
              <FormField hint="Seul votre âge sera visible. Il ne pourra plus être modifié." label="Date de naissance">
                <Input
                  accessibilityLabel="Date de naissance"
                  editable={!hasProfile}
                  keyboardType="number-pad"
                  maxLength={10}
                  onChangeText={(t) => setBirth(formatBirthDateInput(t))}
                  placeholder="JJ/MM/AAAA"
                  value={birth}
                />
              </FormField>
              <FormField label="Je suis">
                <View style={styles.row}>
                  {GENDERS.map((g) => (
                    <Chip key={g.value} label={g.label} onPress={() => setGender(g.value)} selected={gender === g.value} />
                  ))}
                </View>
              </FormField>
            </View>
          ) : null}

          {step === 1 ? (
            <View style={styles.form}>
              <Text accessibilityRole="header" variant="title">
                Qui souhaitez-vous rencontrer ?
              </Text>
              <PreferencesEditor onChange={setPrefs} value={prefs} />
            </View>
          ) : null}

          {step === 2 ? (
            <View style={styles.form}>
              <Text accessibilityRole="header" variant="title">
                Vos plus belles photos
              </Text>
              <Text tone="muted">Au moins une photo, de préférence de face et bien éclairée.</Text>
              <PhotoManager onChanged={(list) => setPhotoCount(list.length)} />
            </View>
          ) : null}

          {step === 3 ? (
            <View style={styles.form}>
              <Text accessibilityRole="header" variant="title">
                Parlez-nous de vous
              </Text>
              <FormField hint={`${bio.length}/500`} label="Bio (facultatif)">
                <Input
                  accessibilityLabel="Bio"
                  maxLength={500}
                  multiline
                  onChangeText={setBio}
                  placeholder="Une phrase, une passion, un défi…"
                  value={bio}
                />
              </FormField>
              <FormField hint="Jusqu'à 10." label="Centres d'intérêt">
                <InterestPicker onChange={setInterests} selected={interests} />
              </FormField>
            </View>
          ) : null}

          {step === 4 ? (
            <View style={styles.form}>
              <Text accessibilityRole="header" variant="title">
                Près de chez vous
              </Text>
              <Text tone="muted">
                Nous utilisons votre position approximative pour vous proposer des profils proches. Les autres ne voient qu&apos;une
                distance arrondie, jamais votre position.
              </Text>
              <Button
                label={location.state === "done" ? "Position enregistrée ✓" : "Partager ma position"}
                loading={location.state === "working"}
                onPress={() => void location.share()}
                variant="secondary"
              />
              {location.message ? <Notice message={location.message} tone={location.state === "done" ? "success" : "warning"} /> : null}
              <FormField hint="Facultatif — affichée sur votre profil." label="Ville">
                <Input accessibilityLabel="Ville" maxLength={80} onChangeText={setCity} value={city} />
              </FormField>
              <Text tone="subtle" variant="caption">
                Sans position, vous voyez des profils sans filtre de distance. Vous pourrez la modifier à tout moment.
              </Text>
            </View>
          ) : null}

          {error ? <Notice message={error} tone="danger" /> : null}

          <View style={styles.actions}>
            <Button label={step === STEPS.length - 1 ? "Commencer à découvrir" : "Continuer"} loading={busy} onPress={next} />
            {step > 0 && step !== 2 ? <Button label="Retour" onPress={() => setStep(step - 1)} variant="ghost" /> : null}
            {step === 0 ? <Button label="Me déconnecter" onPress={() => void signOut()} variant="ghost" /> : null}
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  center: { flex: 1, alignItems: "center", justifyContent: "center" },
  content: { padding: spacing.xl, gap: spacing.xl, maxWidth: 560, width: "100%", alignSelf: "center" },
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: 2 },
  form: { gap: spacing.lg },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  actions: { gap: spacing.sm }
});
