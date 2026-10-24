import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { Gender, Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { parseBirthDate, splitIsoDate } from "../../lib/date";
import { shareApproximateLocation } from "../../lib/location";
import { errorMessage } from "../../lib/validation";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Button, FormError, LoadingState, Screen, Text, TextField, useToast } from "../../ui";
import {
  BirthDateFields,
  DEFAULT_PREFERENCES,
  GenderPicker,
  InterestPicker,
  PreferencesForm,
} from "../profile/forms";
import { PhotoManager } from "../profile/PhotoManager";
import { useOwnProfile } from "../profile/useOwnProfile";

const STEPS = ["Vous", "À propos", "Photos", "Rencontres"] as const;

export function OnboardingScreen() {
  const { colors } = useTheme();
  const { logout, refreshUser } = useAuth();
  const queryClient = useQueryClient();
  const toast = useToast();
  const { data: own, isLoading } = useOwnProfile();

  const [step, setStep] = useState<number | null>(null);
  const [firstName, setFirstName] = useState("");
  const [date, setDate] = useState({ day: "", month: "", year: "" });
  const [gender, setGender] = useState<Gender | null>(null);
  const [bio, setBio] = useState("");
  const [city, setCity] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [prefs, setPrefs] = useState<Preferences>(DEFAULT_PREFERENCES);
  const [shareLocation, setShareLocation] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dateError, setDateError] = useState<string | null>(null);
  const [hydrated, setHydrated] = useState(false);

  // Resume where the user left off: fields come back from the API, and so does the step.
  useEffect(() => {
    if (isLoading || hydrated) return;
    setHydrated(true);
    if (!own) {
      setStep(0);
      return;
    }
    setFirstName(own.firstName);
    setDate(splitIsoDate(own.birthDate));
    setGender(own.gender);
    setBio(own.bio);
    setCity(own.city);
    setInterests(own.interests.map((i) => i.slug));
    if (own.preferences) setPrefs(own.preferences);
    setStep(own.photos.length === 0 ? 2 : 3);
  }, [isLoading, hydrated, own]);

  if (step === null) return <LoadingState />;

  const saveProfile = async (extra: { bio?: string; city?: string; interests?: string[] }) => {
    const parsed = parseBirthDate(date.day, date.month, date.year);
    if (!parsed.ok || !gender) throw new Error("Profil incomplet.");
    await profileApi.save({
      firstName: firstName.trim(),
      birthDate: parsed.iso,
      gender,
      bio: extra.bio ?? bio,
      city: extra.city ?? city,
      interests: extra.interests ?? interests,
    });
    await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
  };

  const next = async () => {
    setError(null);
    setDateError(null);
    setBusy(true);
    try {
      if (step === 0) {
        const parsed = parseBirthDate(date.day, date.month, date.year);
        if (!firstName.trim()) throw new Error("Entrez votre prénom.");
        if (!parsed.ok) {
          setDateError(parsed.error);
          return;
        }
        if (!gender) throw new Error("Choisissez votre genre.");
        await saveProfile({});
        setStep(1);
      } else if (step === 1) {
        await saveProfile({ bio: bio.trim(), city: city.trim(), interests });
        setStep(2);
      } else if (step === 2) {
        if (!own || own.photos.length === 0)
          throw new Error("Ajoutez au moins une photo pour continuer.");
        setStep(3);
      } else {
        if (prefs.interestedIn.length === 0)
          throw new Error("Choisissez au moins un genre à rencontrer.");
        await profileApi.savePreferences(prefs);
        if (shareLocation) {
          const outcome = await shareApproximateLocation();
          if (outcome !== "saved") {
            toast.show({
              message: "Position non partagée : vous verrez des profils sans filtre de distance.",
            });
          }
        }
        await refreshUser(); // profileComplete -> true switches the app to the main navigation
      }
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="onboarding-screen">
      <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((label, i) => (
          <View
            key={label}
            style={[styles.bar, { backgroundColor: i <= step ? colors.primary : colors.border }]}
          />
        ))}
      </View>

      {step === 0 ? (
        <>
          <Text variant="title">Faisons connaissance</Text>
          <TextField
            testID="onboarding-name"
            label="Prénom"
            value={firstName}
            onChangeText={setFirstName}
            autoCapitalize="words"
            maxLength={40}
          />
          <BirthDateFields
            {...date}
            error={dateError}
            onChange={(part, value) => setDate((d) => ({ ...d, [part]: value }))}
            editable={!own}
          />
          {own ? (
            <Text variant="caption" muted>
              La date de naissance ne peut plus être modifiée.
            </Text>
          ) : null}
          <View style={{ gap: spacing.sm }}>
            <Text variant="label" muted>
              Je suis
            </Text>
            <GenderPicker value={gender} onChange={setGender} testIDPrefix="onboarding-gender" />
          </View>
        </>
      ) : null}

      {step === 1 ? (
        <>
          <Text variant="title">Parlez-nous de vous</Text>
          <TextField
            testID="onboarding-bio"
            label="Courte bio"
            hint={`${bio.length}/500`}
            value={bio}
            onChangeText={setBio}
            multiline
            maxLength={500}
          />
          <TextField
            testID="onboarding-city"
            label="Ville"
            value={city}
            onChangeText={setCity}
            maxLength={80}
          />
          <InterestPicker value={interests} onChange={setInterests} />
        </>
      ) : null}

      {step === 2 ? (
        <>
          <Text variant="title">Vos photos</Text>
          <Text muted>Ajoutez jusqu&apos;à 6 photos. La première sera votre photo principale.</Text>
          <PhotoManager photos={own?.photos ?? []} />
        </>
      ) : null}

      {step === 3 ? (
        <>
          <Text variant="title">Qui souhaitez-vous rencontrer ?</Text>
          <PreferencesForm value={prefs} onChange={setPrefs} />
          <Button
            testID="onboarding-location-toggle"
            variant={shareLocation ? "primary" : "secondary"}
            title={
              shareLocation
                ? "Position approximative : activée"
                : "Position approximative : désactivée"
            }
            onPress={() => setShareLocation(!shareLocation)}
          />
          <Text variant="caption" muted>
            Votre position exacte n&apos;est jamais affichée : les autres voient seulement une
            distance approximative.
          </Text>
        </>
      ) : null}

      <FormError testID="onboarding-error" message={error} />
      <View style={styles.footer}>
        <Button
          testID="onboarding-next"
          title={step === STEPS.length - 1 ? "Découvrir des profils" : "Continuer"}
          onPress={next}
          loading={busy}
        />
        {step > 0 ? (
          <Button title="Retour" variant="ghost" onPress={() => setStep(step - 1)} />
        ) : null}
        <Button title="Se déconnecter" variant="ghost" onPress={logout} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: radius.pill },
  footer: { gap: spacing.sm, marginTop: spacing.md },
});
