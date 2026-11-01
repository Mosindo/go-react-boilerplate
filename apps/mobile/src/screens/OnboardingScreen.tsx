import { useState } from "react";
import { View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { GENDER_LABELS, profileApi } from "../api";
import type { Gender, Settings } from "../api/types";
import { useToast } from "../components/feedback";
import { InterestPicker } from "../components/InterestPicker";
import { LocationPicker, type Place } from "../components/LocationPicker";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesForm } from "../components/PreferencesForm";
import { Button, Chip, Screen, Text, TextField } from "../components/ui";
import { maskBirthInput, parseBirthInput, validateBirthInput } from "../lib/date";
import { errorMessage } from "../lib/errors";
import { keys, usePhotos } from "../state/queries";
import { useAuth } from "../state/auth";
import { spacing, useColors } from "../theme";

const STEPS = ["Vous", "Lieu & bio", "Préférences", "Centres d'intérêt", "Photos"] as const;
const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export function OnboardingScreen() {
  const { refreshUser, signOut } = useAuth();
  const toast = useToast();
  const colors = useColors();
  const queryClient = useQueryClient();
  const photos = usePhotos();

  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState(false);
  const [firstName, setFirstName] = useState("");
  const [birth, setBirth] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [bio, setBio] = useState("");
  const [place, setPlace] = useState<Place | null>(null);
  const [interests, setInterests] = useState<string[]>([]);
  const [settings, setSettings] = useState<Settings>({
    interestedIn: ["woman", "man", "nonbinary"],
    minAge: 18,
    maxAge: 60,
    maxDistanceKm: 50,
    discoverable: true,
    showDistance: true,
  });
  const [errors, setErrors] = useState<{ firstName?: string; birth?: string; gender?: string; place?: string }>({});

  const validateStep = (): boolean => {
    if (step === 0) {
      const next = {
        firstName: firstName.trim() ? undefined : "Prénom requis",
        birth: validateBirthInput(birth) ?? undefined,
        gender: gender ? undefined : "Choisissez une option",
      };
      setErrors(next);
      return !next.firstName && !next.birth && !next.gender;
    }
    if (step === 1) {
      const next = { place: place ? undefined : "Indiquez votre localisation" };
      setErrors(next);
      return !next.place;
    }
    return true;
  };

  const finish = async () => {
    const birthDate = parseBirthInput(birth);
    if (!birthDate || !gender || !place) {
      return;
    }
    setBusy(true);
    try {
      await profileApi.saveMine({
        firstName: firstName.trim(),
        birthDate,
        gender,
        bio: bio.trim(),
        city: place.city,
        latitude: place.latitude,
        longitude: place.longitude,
        interests,
      });
      await profileApi.saveSettings(settings);
      queryClient.removeQueries({ queryKey: keys.profile });
      await refreshUser(); // profileComplete=true => bascule vers l'application
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  const isLast = step === STEPS.length - 1;
  const hasPhoto = (photos.data?.length ?? 0) > 0;

  return (
    <Screen scroll>
      <View style={{ flexDirection: "row", gap: 6, marginBottom: spacing.lg }}>
        {STEPS.map((label, index) => (
          <View
            accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length}: ${label}`}
            key={label}
            style={{
              flex: 1,
              height: 4,
              borderRadius: 2,
              backgroundColor: index <= step ? colors.primary : colors.border,
            }}
          />
        ))}
      </View>
      <Text style={{ marginBottom: spacing.lg }} variant="title">
        {STEPS[step]}
      </Text>

      {step === 0 ? (
        <View>
          <TextField
            autoCapitalize="words"
            error={errors.firstName}
            label="Prénom"
            maxLength={40}
            onChangeText={setFirstName}
            value={firstName}
          />
          <TextField
            error={errors.birth}
            keyboardType="number-pad"
            label="Date de naissance (JJ/MM/AAAA)"
            onChangeText={(text) => setBirth(maskBirthInput(text))}
            placeholder="JJ/MM/AAAA"
            value={birth}
          />
          <Text muted style={{ marginBottom: spacing.sm }} variant="label">
            Je suis
          </Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
            {GENDERS.map((g) => (
              <Chip key={g} label={GENDER_LABELS[g]} onPress={() => setGender(g)} selected={gender === g} />
            ))}
          </View>
          {errors.gender ? (
            <Text style={{ color: colors.danger }} variant="caption">
              {errors.gender}
            </Text>
          ) : null}
        </View>
      ) : null}

      {step === 1 ? (
        <View>
          <LocationPicker onChange={setPlace} value={place} />
          {errors.place ? (
            <Text style={{ color: colors.danger, marginTop: spacing.sm }} variant="caption">
              {errors.place}
            </Text>
          ) : null}
          <View style={{ height: spacing.lg }} />
          <TextField
            label="Courte bio (facultatif)"
            maxLength={500}
            multiline
            onChangeText={setBio}
            style={{ minHeight: 96, textAlignVertical: "top" }}
            value={bio}
          />
        </View>
      ) : null}

      {step === 2 ? <PreferencesForm onChange={setSettings} showPrivacy={false} value={settings} /> : null}

      {step === 3 ? (
        <View>
          <Text muted style={{ marginBottom: spacing.md }}>
            Choisissez ce qui vous ressemble. Ils servent à trouver des affinités.
          </Text>
          <InterestPicker onChange={setInterests} selected={interests} />
        </View>
      ) : null}

      {step === 4 ? (
        <View>
          <Text muted style={{ marginBottom: spacing.md }}>
            Ajoutez au moins une photo (6 maximum). Elles ne sont visibles que par les membres connectés.
          </Text>
          <PhotoManager />
        </View>
      ) : null}

      <View style={{ marginTop: spacing.xl }}>
        <Button
          disabled={isLast && !hasPhoto}
          loading={busy}
          onPress={() => {
            if (!validateStep()) {
              return;
            }
            if (isLast) {
              void finish();
            } else {
              setStep(step + 1);
            }
          }}
          testID="onboarding-next"
          title={isLast ? "Terminer" : "Continuer"}
        />
        {step > 0 ? <Button onPress={() => setStep(step - 1)} title="Retour" variant="ghost" /> : null}
        <Button onPress={() => void signOut()} title="Me déconnecter" variant="ghost" />
      </View>
    </Screen>
  );
}
