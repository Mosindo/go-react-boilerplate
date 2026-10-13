import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { saveLocation, savePreferences, saveProfile } from "../api/platform";
import { errorMessage } from "../api/client";
import type { Gender, OwnProfile, Preferences } from "../api/types";
import { DEFAULT_PREFERENCES, PreferencesFields } from "../features/profile/PreferencesFields";
import { InterestPicker } from "../features/profile/InterestPicker";
import { PhotoGrid } from "../features/profile/PhotoGrid";
import { useOwnProfile } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { GENDERS, GENDER_LABELS, formatBirthDate, isAdult, parseBirthDateInput } from "../lib/format";
import { getApproximateLocation } from "../lib/location";
import { queryKeys } from "../lib/queryClient";
import { validateFirstName } from "../lib/validation";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

const STEPS = ["Vous", "Vos envies", "Photos", "À propos", "Localisation"] as const;

type OnboardingScreenProps = {
  onDone: () => void;
};

export default function OnboardingScreen({ onDone }: OnboardingScreenProps) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const profileQuery = useOwnProfile();
  const existing: OwnProfile | null = profileQuery.data ?? null;

  const [step, setStep] = useState(0);
  const [firstName, setFirstName] = useState(existing?.firstName ?? "");
  const [birthInput, setBirthInput] = useState(existing ? formatBirthDate(existing.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(existing?.gender ?? null);
  const [prefs, setPrefs] = useState<Preferences>(existing?.preferences ?? DEFAULT_PREFERENCES);
  const [bio, setBio] = useState(existing?.bio ?? "");
  const [city, setCity] = useState(existing?.city ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(existing?.interestIds ?? []);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [locationStatus, setLocationStatus] = useState<string | null>(null);

  const photos = profileQuery.data?.photos ?? [];

  const buildProfileInput = () => ({
    firstName: firstName.trim(),
    birthDate: parseBirthDateInput(birthInput) ?? "",
    gender: gender as Gender,
    bio: bio.trim(),
    city: city.trim() || null,
    interestIds
  });

  const save = useMutation({
    mutationFn: async (target: number) => {
      if (target === 0) {
        await saveProfile(buildProfileInput());
      } else if (target === 1) {
        await savePreferences(prefs);
      } else if (target === 3) {
        await saveProfile(buildProfileInput());
      }
    },
    onSuccess: async (_data, target) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      setStep(target + 1);
    },
    onError: (error) => setFormError(errorMessage(error))
  });

  const locate = useMutation({
    mutationFn: async () => {
      const result = await getApproximateLocation();
      if (result.status !== "ok") {
        return result.status;
      }
      await saveLocation({ latitude: result.latitude, longitude: result.longitude });
      if (!city.trim() && result.city) {
        setCity(result.city);
        await saveProfile({ ...buildProfileInput(), city: result.city });
      }
      return "ok" as const;
    },
    onSuccess: async (status) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      if (status === "ok") {
        onDone();
      } else if (status === "denied") {
        setLocationStatus("Localisation refusée. Vous pourrez l'activer plus tard dans votre profil.");
      } else {
        setLocationStatus("Position introuvable pour le moment. Vous pourrez réessayer depuis votre profil.");
      }
    },
    onError: (error) => setLocationStatus(errorMessage(error))
  });

  function next() {
    setFormError(null);
    const nextErrors: Record<string, string> = {};
    if (step === 0) {
      const nameError = validateFirstName(firstName);
      if (nameError) {
        nextErrors.firstName = nameError;
      }
      const iso = parseBirthDateInput(birthInput);
      if (!iso) {
        nextErrors.birth = "Format attendu : JJ/MM/AAAA.";
      } else if (!isAdult(iso)) {
        nextErrors.birth = "Vous devez avoir au moins 18 ans pour utiliser l'application.";
      }
      if (!gender) {
        nextErrors.gender = "Choisissez une option.";
      }
    }
    if (step === 1 && prefs.interestedIn.length === 0) {
      nextErrors.interestedIn = "Choisissez au moins une option.";
    }
    if (step === 2 && photos.length === 0) {
      nextErrors.photos = "Ajoutez au moins une photo pour continuer.";
    }
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) {
      return;
    }
    if (step === 0 || step === 1 || step === 3) {
      save.mutate(step);
    } else {
      setStep(step + 1);
    }
  }

  const busy = save.isPending || locate.isPending;
  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <View style={styles.progress}>
          {STEPS.map((label, index) => (
            <View
              accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length} : ${label}`}
              key={label}
              style={[styles.bar, { backgroundColor: index <= step ? colors.primary : colors.border }]}
            />
          ))}
        </View>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <View style={styles.content}>
            <Text tone="primary" variant="eyebrow" weight="bold">
              Étape {step + 1} sur {STEPS.length}
            </Text>
            {formError ? <Notice message={formError} tone="danger" /> : null}

            {step === 0 ? (
              <View style={styles.section}>
                <Text accessibilityRole="header" variant="title" weight="bold">
                  Faisons connaissance
                </Text>
                <Input
                  autoCapitalize="words"
                  error={errors.firstName}
                  label="Prénom"
                  onChangeText={setFirstName}
                  testID="onboarding-name"
                  value={firstName}
                />
                <Input
                  error={errors.birth}
                  hint="Votre âge est affiché, jamais votre date de naissance. Elle ne pourra plus être modifiée."
                  keyboardType="numbers-and-punctuation"
                  label="Date de naissance"
                  maxLength={10}
                  onChangeText={setBirthInput}
                  placeholder="JJ/MM/AAAA"
                  testID="onboarding-birth"
                  value={birthInput}
                />
                <FormField error={errors.gender} label="Je suis">
                  <View style={styles.wrap}>
                    {GENDERS.map((g) => (
                      <Chip key={g} label={GENDER_LABELS[g]} onPress={() => setGender(g)} selected={gender === g} testID={`gender-${g}`} />
                    ))}
                  </View>
                </FormField>
              </View>
            ) : null}

            {step === 1 ? (
              <View style={styles.section}>
                <Text accessibilityRole="header" variant="title" weight="bold">
                  Qui cherchez-vous ?
                </Text>
                <PreferencesFields error={errors.interestedIn} onChange={setPrefs} value={prefs} />
              </View>
            ) : null}

            {step === 2 ? (
              <View style={styles.section}>
                <Text accessibilityRole="header" variant="title" weight="bold">
                  Vos plus belles photos
                </Text>
                <Text tone="muted">Au moins une photo. Elles ne sont visibles que par les membres connectés.</Text>
                <PhotoGrid photos={photos} />
                {errors.photos ? <Notice message={errors.photos} tone="warning" /> : null}
              </View>
            ) : null}

            {step === 3 ? (
              <View style={styles.section}>
                <Text accessibilityRole="header" variant="title" weight="bold">
                  Parlez-nous de vous
                </Text>
                <Input
                  hint={`${bio.length}/500`}
                  label="Courte bio (facultatif)"
                  maxLength={500}
                  multiline
                  onChangeText={setBio}
                  testID="onboarding-bio"
                  value={bio}
                />
                <Input label="Ville (facultatif)" maxLength={80} onChangeText={setCity} value={city} />
                <FormField hint="Jusqu'à 10." label="Centres d'intérêt">
                  <InterestPicker onChange={setInterestIds} value={interestIds} />
                </FormField>
              </View>
            ) : null}

            {step === 4 ? (
              <View style={styles.section}>
                <Text accessibilityRole="header" variant="title" weight="bold">
                  Autour de vous
                </Text>
                <Text tone="muted">
                  Votre position sert uniquement à trouver des profils proches. Elle est arrondie à environ 1 km, et les autres ne voient
                  qu'une distance approximative.
                </Text>
                {locationStatus ? <Notice message={locationStatus} tone="warning" /> : null}
                <Button
                  fullWidth
                  label="Utiliser ma position"
                  loading={locate.isPending}
                  onPress={() => locate.mutate()}
                  size="lg"
                  testID="onboarding-locate"
                />
                <Button fullWidth label="Plus tard" onPress={onDone} variant="ghost" testID="onboarding-skip-location" />
              </View>
            ) : null}
          </View>
        </ScrollView>

        {step < 4 ? (
          <View style={[styles.footer, { borderColor: colors.border, backgroundColor: colors.background }]}>
            {step > 0 ? (
              <Button disabled={busy} label="Retour" onPress={() => setStep(step - 1)} variant="outline" />
            ) : (
              <Button disabled={busy} label="Se déconnecter" onPress={() => void signOut()} variant="ghost" />
            )}
            <Button label="Continuer" loading={save.isPending} onPress={next} testID="onboarding-next" />
          </View>
        ) : null}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  progress: { flexDirection: "row", gap: spacing.xs, paddingHorizontal: spacing.lg, paddingTop: spacing.md },
  bar: { flex: 1, height: 4, borderRadius: radii.pill },
  scroll: { flexGrow: 1, padding: spacing.lg },
  content: { width: "100%", maxWidth: 520, alignSelf: "center", gap: spacing.lg, paddingBottom: spacing.xl },
  section: { gap: spacing.lg },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  footer: {
    flexDirection: "row",
    justifyContent: "space-between",
    padding: spacing.lg,
    borderTopWidth: StyleSheet.hairlineWidth
  }
});
