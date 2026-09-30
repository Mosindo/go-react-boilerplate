import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { Button, Chip, ErrorState, LoadingState, Screen, Text, TextField } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { profileApi } from "../../lib/api/endpoints";
import type { Gender, OwnProfile, RelationshipGoal } from "../../lib/api/types";
import { ageFromIso, genderLabels, goalLabels, interestedInLabels, maskDateInput, parseFrenchDate } from "../../lib/format";
import { useSession } from "../../lib/session/SessionProvider";
import { InterestPicker } from "../profile/InterestPicker";
import { LocationControl } from "../profile/LocationControl";
import { PhotoGrid } from "../profile/PhotoGrid";
import { useProfile, useProfileMutation } from "../profile/hooks";
import { useOnboarding } from "./OnboardingContext";

type StepKey = "name" | "birthdate" | "gender" | "seeking" | "photos" | "location" | "about";

const STEPS: StepKey[] = ["name", "birthdate", "gender", "seeking", "photos", "location", "about"];
const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

function firstIncompleteStep(profile: OwnProfile): number {
  if (!profile.firstName) return 0;
  if (!profile.birthdate) return 1;
  if (!profile.gender) return 2;
  if (profile.preferences.interestedIn.length === 0) return 3;
  if (profile.photos.length === 0) return 4;
  return 5;
}

export default function OnboardingScreen() {
  const { data: profile, isLoading, error, refetch } = useProfile();
  if (isLoading) return <LoadingState label="Préparation de votre profil…" />;
  if (error || !profile) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;
  return <OnboardingFlow profile={profile} />;
}

function OnboardingFlow({ profile }: { profile: OwnProfile }) {
  const { colors, radii } = useTheme();
  const { signOut } = useSession();
  const { finish } = useOnboarding();
  const [step, setStep] = useState(() => firstIncompleteStep(profile));
  const [firstName, setFirstName] = useState(profile.firstName);
  const [birthdate, setBirthdate] = useState("");
  const [gender, setGender] = useState<Gender | null>(profile.gender);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile.preferences.interestedIn);
  const [goal, setGoal] = useState<RelationshipGoal | null>(profile.relationshipGoal);
  const [bio, setBio] = useState(profile.bio);
  const [interests, setInterests] = useState<number[]>(profile.interests.map((i) => i.id));
  const [error, setError] = useState<string | null>(null);

  const update = useProfileMutation(profileApi.update);
  const updatePrefs = useProfileMutation(profileApi.updatePreferences);
  const updateInterests = useProfileMutation(profileApi.updateInterests);
  const saving = update.isPending || updatePrefs.isPending || updateInterests.isPending;

  const key = STEPS[step];
  const isoBirthdate = useMemo(() => parseFrenchDate(birthdate), [birthdate]);

  const next = async () => {
    setError(null);
    try {
      switch (key) {
        case "name":
          if (!firstName.trim()) return setError("Indiquez votre prénom.");
          await update.mutateAsync({ firstName: firstName.trim() });
          break;
        case "birthdate":
          if (!isoBirthdate) return setError("Format attendu : JJ/MM/AAAA.");
          if (ageFromIso(isoBirthdate) < 18) return setError("Vous devez avoir au moins 18 ans pour utiliser Lueur.");
          await update.mutateAsync({ birthdate: isoBirthdate });
          break;
        case "gender":
          if (!gender) return setError("Choisissez une option.");
          await update.mutateAsync({ gender });
          break;
        case "seeking":
          if (interestedIn.length === 0) return setError("Choisissez au moins une option.");
          await updatePrefs.mutateAsync({ ...profile.preferences, interestedIn });
          await update.mutateAsync({ relationshipGoal: goal ?? "" });
          break;
        case "photos":
          if (profile.photos.length === 0) return setError("Ajoutez au moins une photo.");
          break;
        case "location":
          break;
        case "about":
          await update.mutateAsync({ bio });
          await updateInterests.mutateAsync(interests);
          finish();
          return;
      }
      setStep((s) => Math.min(s + 1, STEPS.length - 1));
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const titles: Record<StepKey, [string, string]> = {
    name: ["Comment vous appelez-vous ?", "Votre prénom sera visible sur votre profil."],
    birthdate: ["Votre date de naissance", "Seul votre âge est affiché. Elle ne pourra plus être modifiée ensuite."],
    gender: ["Vous êtes…", "Pour vous présenter aux bonnes personnes."],
    seeking: ["Qui souhaitez-vous rencontrer ?", "Vous pourrez changer cela à tout moment."],
    photos: ["Ajoutez vos photos", "Au moins une photo, jusqu'à six. La première sera votre photo principale."],
    location: ["Où êtes-vous ?", "Facultatif. Sans position, votre profil ne sera proposé qu’aux membres qui n’ont pas non plus partagé la leur."],
    about: ["Parlez un peu de vous", "Une courte bio et quelques centres d'intérêt aident à briser la glace."]
  };
  const [title, subtitle] = titles[key];

  return (
    <Screen
      footer={
        <View style={styles.footer}>
          {step > 0 ? <Button label="Retour" onPress={() => setStep((s) => s - 1)} variant="ghost" /> : <Button label="Se déconnecter" onPress={() => void signOut()} variant="ghost" />}
          <Button
            label={key === "about" ? "Terminer" : key === "location" && !profile.hasLocation ? "Passer" : "Continuer"}
            loading={saving}
            onPress={next}
            style={styles.primary}
            testID="onboarding-next"
          />
        </View>
      }
      scroll
      testID={`onboarding-step-${key}`}
    >
      <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={[styles.bar, { borderRadius: radii.pill, backgroundColor: i <= step ? colors.primary : colors.border }]} />
        ))}
      </View>
      <Text variant="title">{title}</Text>
      <Text tone="muted">{subtitle}</Text>

      {key === "name" ? (
        <TextField autoComplete="given-name" autoFocus label="Prénom" maxLength={40} onChangeText={setFirstName} onSubmitEditing={next} testID="onboarding-firstname" value={firstName} />
      ) : null}

      {key === "birthdate" ? (
        <TextField
          autoFocus
          hint="JJ/MM/AAAA"
          keyboardType="number-pad"
          label="Date de naissance"
          onChangeText={(v) => setBirthdate(maskDateInput(v))}
          placeholder="JJ/MM/AAAA"
          testID="onboarding-birthdate"
          value={birthdate}
        />
      ) : null}

      {key === "gender" ? (
        <View style={styles.options}>
          {GENDERS.map((g) => (
            <Chip key={g} label={genderLabels[g]} onPress={() => setGender(g)} selected={gender === g} testID={`gender-${g}`} />
          ))}
        </View>
      ) : null}

      {key === "seeking" ? (
        <>
          <View style={styles.options}>
            {GENDERS.map((g) => (
              <Chip
                key={g}
                label={interestedInLabels[g]}
                onPress={() => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]))}
                selected={interestedIn.includes(g)}
                testID={`seeking-${g}`}
              />
            ))}
          </View>
          <Text variant="label">Ce que vous recherchez (facultatif)</Text>
          <View style={styles.options}>
            {(Object.keys(goalLabels) as RelationshipGoal[]).map((g) => (
              <Chip key={g} label={goalLabels[g]} onPress={() => setGoal(goal === g ? null : g)} selected={goal === g} />
            ))}
          </View>
        </>
      ) : null}

      {key === "photos" ? <PhotoGrid photos={profile.photos} /> : null}
      {key === "location" ? <LocationControl profile={profile} /> : null}

      {key === "about" ? (
        <>
          <TextField label="Bio" maxLength={500} multiline onChangeText={setBio} placeholder="Ce qui vous fait vibrer, ce que vous aimez partager…" testID="onboarding-bio" value={bio} />
          <Text variant="label">Centres d’intérêt</Text>
          <InterestPicker onChange={setInterests} selected={interests} />
        </>
      ) : null}

      {error ? (
        <Text accessibilityLiveRegion="polite" testID="onboarding-error" tone="danger">
          {error}
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: 6, marginBottom: 8 },
  bar: { flex: 1, height: 4 },
  options: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  footer: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: 12 },
  primary: { flex: 1, maxWidth: 260 }
});
