import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { describeError } from "../api/client";
import { keys } from "../api/keys";
import { saveLocation } from "../api/platform";
import type { Gender, LookingFor, Me, Preferences } from "../api/types";
import { ChoiceGroup, GENDER_OPTIONS, InterestPicker, LocationField, PreferencesFields, type Option } from "../components/Pickers";
import { PhotoManager } from "../components/PhotoManager";
import { useSavePreferences, useSaveProfile } from "../hooks/data";
import { useAuth } from "../hooks/useAuth";
import { LOOKING_FOR_LABELS } from "../shared/format";
import { SafeAreaLayout, ScreenContainer } from "../shared/layout";
import { Button, FormField, Notice, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";
import { digitsOnly, parseBirthdate, splitBirthdate, validateFirstName, type BirthdateParts } from "../utils/validation";

const LOOKING_OPTIONS: Option<LookingFor>[] = (Object.keys(LOOKING_FOR_LABELS) as LookingFor[]).map((value) => ({ value, label: LOOKING_FOR_LABELS[value] }));
const STEP_TITLES = ["Vous", "Vos envies", "Votre profil", "Vos photos"] as const;

/** Four-step profile creation shown until the account has a profile and at least one photo. */
export default function OnboardingScreen({ me, onFinish }: { me: Me; onFinish?: () => void }) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const client = useQueryClient();
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();

  const existing = me.profile;
  const [step, setStep] = useState(existing ? 3 : 0);
  const [firstName, setFirstName] = useState(existing?.firstName ?? "");
  const [birth, setBirth] = useState<BirthdateParts>(existing ? splitBirthdate(existing.birthdate) : { day: "", month: "", year: "" });
  const [gender, setGender] = useState<Gender[]>(existing ? [existing.gender] : []);
  const [prefs, setPrefs] = useState<Preferences>(me.preferences);
  const [bio, setBio] = useState(existing?.bio ?? "");
  const [occupation, setOccupation] = useState(existing?.occupation ?? "");
  const [lookingFor, setLookingFor] = useState<LookingFor[]>([existing?.lookingFor ?? "open"]);
  const [interests, setInterests] = useState<string[]>(existing?.interests.map((i) => i.slug) ?? []);
  const [city, setCity] = useState(existing?.city ?? "");
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const birthResult = useMemo(() => parseBirthdate(birth), [birth]);
  const nameError = validateFirstName(firstName);

  const validStep0 = !nameError && birthResult.error === null && gender.length === 1;

  const next = () => {
    setTouched(true);
    setError(null);
    if (step === 0 && !validStep0) return;
    setTouched(false);
    setStep((s) => s + 1);
  };

  /** Persists everything entered so far; photos (step 4) are uploaded as the user adds them. */
  const saveAll = async () => {
    if (birthResult.iso === null || gender.length !== 1) return false;
    setBusy(true);
    setError(null);
    try {
      await saveProfile.mutateAsync({
        firstName: firstName.trim(),
        birthdate: birthResult.iso,
        gender: gender[0],
        bio,
        occupation,
        lookingFor: lookingFor[0],
        city,
        interests
      });
      await savePreferences.mutateAsync(prefs);
      if (coords) client.setQueryData(keys.me, await saveLocation(coords.latitude, coords.longitude, city));
      return true;
    } catch (e) {
      setError(describeError(e));
      return false;
    } finally {
      setBusy(false);
    }
  };

  const toPhotos = async () => {
    if (await saveAll()) setStep(3);
  };

  const finish = async () => {
    // Refresh so the root switches to the main app once profile + photo exist.
    await client.invalidateQueries({ queryKey: keys.me });
    onFinish?.();
  };

  const hasPhoto = me.photos.length > 0;

  return (
    <SafeAreaLayout>
      <ScreenContainer
        footer={
          <View style={styles.footer}>
            {step > 0 && step < 3 ? (
              <Button onPress={() => setStep((s) => s - 1)} variant="ghost">
                Retour
              </Button>
            ) : null}
            {step < 2 ? (
              <Button onPress={next} size="lg" style={styles.grow}>
                Continuer
              </Button>
            ) : step === 2 ? (
              <Button loading={busy} onPress={toPhotos} size="lg" style={styles.grow}>
                Continuer
              </Button>
            ) : (
              <Button disabled={!hasPhoto} onPress={finish} size="lg" style={styles.grow}>
                Découvrir des profils
              </Button>
            )}
          </View>
        }
      >
        <View accessibilityLabel={`Étape ${step + 1} sur 4`} style={styles.progress}>
          {STEP_TITLES.map((title, index) => (
            <View key={title} style={[styles.bar, { backgroundColor: index <= step ? colors.primary : colors.border }]} />
          ))}
        </View>
        <Text accessibilityRole="header" variant="title">
          {STEP_TITLES[step]}
        </Text>
        {error ? <Notice message={error} tone="error" /> : null}

        {step === 0 ? (
          <View style={styles.block}>
            <Text tone="muted">Quelques informations pour commencer. Votre âge sera affiché, pas votre date de naissance.</Text>
            <FormField autoCapitalize="words" error={touched ? nameError : null} label="Prénom" maxLength={40} onChangeText={setFirstName} value={firstName} />
            <View>
              <Text style={styles.label} tone="muted" variant="label">
                Date de naissance
              </Text>
              <View style={styles.dateRow}>
                <View style={styles.grow}>
                  <FormField
                    accessibilityLabel="Jour de naissance"
                    keyboardType="number-pad"
                    label="Jour"
                    maxLength={2}
                    onChangeText={(v) => setBirth({ ...birth, day: digitsOnly(v, 2) })}
                    placeholder="JJ"
                    value={birth.day}
                  />
                </View>
                <View style={styles.grow}>
                  <FormField
                    accessibilityLabel="Mois de naissance"
                    keyboardType="number-pad"
                    label="Mois"
                    maxLength={2}
                    onChangeText={(v) => setBirth({ ...birth, month: digitsOnly(v, 2) })}
                    placeholder="MM"
                    value={birth.month}
                  />
                </View>
                <View style={styles.grow}>
                  <FormField
                    accessibilityLabel="Année de naissance"
                    keyboardType="number-pad"
                    label="Année"
                    maxLength={4}
                    onChangeText={(v) => setBirth({ ...birth, year: digitsOnly(v, 4) })}
                    placeholder="AAAA"
                    value={birth.year}
                  />
                </View>
              </View>
              {touched && birthResult.error ? (
                <Text tone="danger" variant="caption">
                  {birthResult.error}
                </Text>
              ) : null}
            </View>
            <Text variant="label" tone="muted">
              Je suis
            </Text>
            <ChoiceGroup onChange={setGender} options={GENDER_OPTIONS} value={gender} />
            {touched && gender.length === 0 ? (
              <Text tone="danger" variant="caption">
                Choisissez une option.
              </Text>
            ) : null}
          </View>
        ) : null}

        {step === 1 ? (
          <View style={styles.block}>
            <Text tone="muted">Ces préférences filtrent les profils que vous voyez. Vous pourrez les modifier à tout moment.</Text>
            <PreferencesFields onChange={setPrefs} value={prefs} />
          </View>
        ) : null}

        {step === 2 ? (
          <View style={styles.block}>
            <FormField counter={500} hint="Quelques mots sur vous." label="À propos de moi" maxLength={500} multiline onChangeText={setBio} value={bio} />
            <FormField label="Profession" maxLength={80} onChangeText={setOccupation} value={occupation} />
            <Text variant="label" tone="muted">
              Ce que je recherche
            </Text>
            <ChoiceGroup onChange={setLookingFor} options={LOOKING_OPTIONS} value={lookingFor} />
            <Text variant="label" tone="muted">
              Centres d'intérêt
            </Text>
            <InterestPicker onChange={setInterests} selected={interests} />
            <LocationField
              city={city}
              hasLocation={coords !== null || Boolean(existing?.hasLocation)}
              onCityChange={setCity}
              onLocated={(location) => {
                setCoords({ latitude: location.latitude, longitude: location.longitude });
                if (location.city) setCity(location.city);
              }}
            />
          </View>
        ) : null}

        {step === 3 ? (
          <View style={styles.block}>
            <Text tone="muted">Ajoutez au moins une photo. La première est votre photo principale ; touchez une photo pour la modifier.</Text>
            <PhotoManager photos={me.photos} />
            <Button onPress={() => void signOut()} size="sm" variant="ghost">
              Me déconnecter
            </Button>
          </View>
        ) : null}
      </ScreenContainer>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  footer: { flexDirection: "row", gap: spacing.sm, width: "100%", maxWidth: 520 },
  grow: { flex: 1 },
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: 2 },
  block: { gap: spacing.lg },
  label: { marginBottom: spacing.xs },
  dateRow: { flexDirection: "row", gap: spacing.sm }
});
