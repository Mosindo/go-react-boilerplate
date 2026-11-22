import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { updatePreferences, updateProfile } from "../../api/dating";
import type { Gender, OwnProfile, ProfilePatch } from "../../api/types";
import { BirthdateField, type BirthdateParts } from "../../components/BirthdateField";
import { ChoiceList } from "../../components/ChoiceList";
import { InterestPicker } from "../../components/InterestPicker";
import { LocationField } from "../../components/LocationField";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useInterests, useMyProfile, useProfileMutation } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { genderLabels, interestedInLabels } from "../../lib/format";
import { firstNameProblem, parseBirthdate } from "../../lib/validation";
import type { MainScreenProps } from "../../navigation/types";
import { Button, LoadingState, Screen, Text, TextField, colors, radii, spacing } from "../../ui";

const STEPS = ["name", "birthdate", "gender", "interestedIn", "photos", "location", "about"] as const;
type Step = (typeof STEPS)[number];

/** Identity basics are mandatory before entering the app. */
export function needsOnboarding(p: OwnProfile): boolean {
  return !p.firstName || !p.birthdate || !p.gender || p.preferences.interestedIn.length === 0;
}

function firstMissingStep(p: OwnProfile): number {
  const order: [Step, boolean][] = [
    ["name", !p.firstName],
    ["birthdate", !p.birthdate],
    ["gender", !p.gender],
    ["interestedIn", p.preferences.interestedIn.length === 0],
    ["photos", p.photos.length === 0],
    ["location", !p.hasLocation]
  ];
  const missing = order.find(([, isMissing]) => isMissing);
  return missing ? STEPS.indexOf(missing[0]) : STEPS.indexOf("about");
}

const titles: Record<Step, [string, string]> = {
  name: ["Comment vous appelez-vous ?", "Votre prénom sera visible sur votre profil."],
  birthdate: ["Votre date de naissance", "Lueur est réservé aux personnes majeures."],
  gender: ["Vous êtes…", "Choisissez ce qui vous correspond."],
  interestedIn: ["Vous souhaitez rencontrer…", "Plusieurs choix possibles. Modifiable à tout moment."],
  photos: ["Ajoutez vos photos", "Au moins une photo, idéalement trois où l'on voit bien votre visage."],
  location: ["Où vous trouvez-vous ?", "Pour vous proposer des personnes proches de vous."],
  about: ["Parlez un peu de vous", "Une bio et quelques centres d'intérêt facilitent le premier message."]
};

export default function OnboardingScreen({ navigation }: MainScreenProps<"Onboarding">) {
  const { data: profile } = useMyProfile();
  const { signOut } = useAuth();
  const interests = useInterests();
  const [stepIndex, setStepIndex] = useState(() => (profile ? firstMissingStep(profile) : 0));
  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [birth, setBirth] = useState<BirthdateParts>(() => {
    const [year = "", month = "", day = ""] = (profile?.birthdate ?? "").split("-");
    return { day, month, year };
  });
  const [gender, setGender] = useState<Gender | null>(profile?.gender ?? null);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile?.preferences.interestedIn ?? []);
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [slugs, setSlugs] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);
  const [error, setError] = useState<string | null>(null);

  const patch = useProfileMutation((p: ProfilePatch) => updateProfile(p));
  const prefs = useProfileMutation(updatePreferences);
  const step = STEPS[stepIndex];
  const progress = useMemo(() => (stepIndex + 1) / STEPS.length, [stepIndex]);

  if (!profile) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }

  const finish = () => navigation.reset({ index: 0, routes: [{ name: "Tabs" }] });
  const next = () => {
    setError(null);
    if (stepIndex === STEPS.length - 1) {
      finish();
    } else {
      setStepIndex(stepIndex + 1);
    }
  };

  const submit = async () => {
    setError(null);
    try {
      switch (step) {
        case "name": {
          const problem = firstNameProblem(firstName);
          if (problem) {
            setError(problem);
            return;
          }
          await patch.mutateAsync({ firstName: firstName.trim() });
          break;
        }
        case "birthdate": {
          const parsed = parseBirthdate(birth.day, birth.month, birth.year);
          if (!parsed.iso) {
            setError(parsed.error ?? "Date invalide.");
            return;
          }
          if (parsed.iso !== profile.birthdate) {
            await patch.mutateAsync({ birthdate: parsed.iso });
          }
          break;
        }
        case "gender":
          if (!gender) {
            setError("Choisissez une option.");
            return;
          }
          await patch.mutateAsync({ gender });
          break;
        case "interestedIn":
          if (interestedIn.length === 0) {
            setError("Choisissez au moins une option.");
            return;
          }
          await prefs.mutateAsync({ ...profile.preferences, interestedIn });
          break;
        case "photos":
          if (profile.photos.length === 0) {
            setError("Ajoutez au moins une photo pour continuer.");
            return;
          }
          break;
        case "location":
          if (!profile.hasLocation) {
            setError("Indiquez votre position pour découvrir des profils proches.");
            return;
          }
          break;
        case "about":
          await patch.mutateAsync({ bio, interests: slugs });
          break;
      }
      next();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const [title, subtitle] = titles[step];
  const saving = patch.isPending || prefs.isPending;

  return (
    <Screen
      footer={
        <>
          <Button label={step === "about" ? "Terminer" : "Continuer"} loading={saving} onPress={submit} testID="onboarding-next" />
          {step === "about" ? <Button label="Plus tard" onPress={finish} testID="onboarding-skip" variant="ghost" /> : null}
        </>
      }
      scroll
      testID={`onboarding-${step}`}
    >
      <View style={styles.top}>
        <View style={styles.progressTrack}>
          <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
        </View>
        <View style={styles.topRow}>
          {stepIndex > 0 ? (
            <Button compact label="Retour" onPress={() => setStepIndex(stepIndex - 1)} variant="ghost" />
          ) : (
            <Button compact label="Se déconnecter" onPress={() => void signOut()} variant="ghost" />
          )}
          <Text tone="subtle" variant="caption">
            {stepIndex + 1} / {STEPS.length}
          </Text>
        </View>
      </View>
      <View style={styles.titles}>
        <Text variant="title">{title}</Text>
        <Text tone="muted">{subtitle}</Text>
      </View>

      {step === "name" ? (
        <TextField
          autoCapitalize="words"
          autoComplete="given-name"
          autoFocus
          label="Prénom"
          maxLength={40}
          onChangeText={setFirstName}
          onSubmitEditing={submit}
          testID="onboarding-firstname"
          value={firstName}
        />
      ) : null}
      {step === "birthdate" ? <BirthdateField onChange={setBirth} value={birth} /> : null}
      {step === "gender" ? (
        <ChoiceList
          onToggle={(g) => setGender(g)}
          options={(Object.keys(genderLabels) as Gender[]).map((g) => ({ value: g, label: genderLabels[g] }))}
          selected={gender ? [gender] : []}
          testIDPrefix="gender"
        />
      ) : null}
      {step === "interestedIn" ? (
        <ChoiceList
          multiple
          onToggle={(g) => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]))}
          options={(Object.keys(interestedInLabels) as Gender[]).map((g) => ({ value: g, label: interestedInLabels[g] }))}
          selected={interestedIn}
          testIDPrefix="interested"
        />
      ) : null}
      {step === "photos" ? <PhotoGrid photos={profile.photos} /> : null}
      {step === "location" ? <LocationField city={profile.city} hasLocation={profile.hasLocation} /> : null}
      {step === "about" ? (
        <View style={styles.about}>
          <TextField
            label="Bio"
            maxLength={500}
            multiline
            onChangeText={setBio}
            placeholder="Ce qui vous fait vibrer, votre dimanche idéal…"
            testID="onboarding-bio"
            value={bio}
          />
          {interests.data ? <InterestPicker catalog={interests.data} onChange={setSlugs} selected={slugs} /> : null}
        </View>
      ) : null}

      {error ? (
        <Text accessibilityLiveRegion="polite" style={styles.error} testID="onboarding-error" tone="danger">
          {error}
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: {
    gap: spacing.xs,
    marginBottom: spacing.lg
  },
  topRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between"
  },
  progressTrack: {
    height: 4,
    borderRadius: radii.pill,
    backgroundColor: colors.surfaceRaised,
    overflow: "hidden"
  },
  progressFill: {
    height: 4,
    backgroundColor: colors.accent
  },
  titles: {
    gap: spacing.xs,
    marginBottom: spacing.xl
  },
  about: {
    gap: spacing.lg
  },
  error: {
    marginTop: spacing.md
  }
});
