import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, LookingFor, MyProfile, ProfileInput } from "../../api/types";
import { keys } from "../../state/queryClient";
import { useMyProfile, useSetMyProfile } from "../../state/queries";
import { useAuth } from "../../state/AuthProvider";
import { radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, Icon, IconButton, InlineError, LoadingState, Screen, Text, TextField } from "../../ui";
import { genderLabels, genders, interestedInLabels, lookingForLabels, lookingFors } from "../../utils/labels";
import { isValidFirstName, parseBirthdate } from "../../utils/validation";
import { BirthdateField, splitBirthdate, type BirthParts } from "../profile/BirthdateField";
import { InterestPicker } from "../profile/InterestPicker";
import { readApproximateLocation } from "../profile/media";
import { PhotoGrid } from "../profile/PhotoGrid";

const STEPS = ["name", "birth", "gender", "seeking", "photos", "about", "location"] as const;
type Step = (typeof STEPS)[number];

type Draft = {
  firstName: string;
  birth: BirthParts;
  gender: Gender | null;
  interestedIn: Gender[];
  lookingFor: LookingFor;
  bio: string;
  jobTitle: string;
  interests: string[];
};

function draftFrom(data: MyProfile | undefined): Draft {
  const p = data?.profile;
  return {
    firstName: p?.firstName ?? "",
    birth: splitBirthdate(p?.birthdate),
    gender: p?.gender ?? null,
    interestedIn: data?.preferences?.interestedIn ?? [],
    lookingFor: p?.lookingFor ?? "not_sure",
    bio: p?.bio ?? "",
    jobTitle: p?.jobTitle ?? "",
    interests: p?.interests.map((i) => i.slug) ?? []
  };
}

export function OnboardingScreen({ onFinished }: { onFinished: () => void }) {
  const { data, isLoading } = useMyProfile();
  if (isLoading || !data) return <LoadingState />;
  return <Wizard initial={data} onFinished={onFinished} />;
}

function Wizard({ initial, onFinished }: { initial: MyProfile; onFinished: () => void }) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const qc = useQueryClient();
  const setMyProfile = useSetMyProfile();
  const { data } = useMyProfile();
  const [draft, setDraft] = useState<Draft>(() => draftFrom(initial));
  const [step, setStep] = useState<Step>(initial.profile ? "photos" : "name");
  const [touched, setTouched] = useState(false);
  const [locationNote, setLocationNote] = useState<string | null>(null);
  const index = STEPS.indexOf(step);
  const photos = data?.photos ?? [];
  const update = (patch: Partial<Draft>) => setDraft((d) => ({ ...d, ...patch }));

  const birth = useMemo(() => parseBirthdate(draft.birth.day, draft.birth.month, draft.birth.year), [draft.birth]);

  const toInput = (): ProfileInput => ({
    firstName: draft.firstName.trim(),
    birthdate: "iso" in birth ? birth.iso : "",
    gender: draft.gender ?? "woman",
    bio: draft.bio,
    jobTitle: draft.jobTitle,
    heightCm: initial.profile?.heightCm ?? null,
    lookingFor: draft.lookingFor,
    interests: draft.interests
  });

  const saveProfile = useMutation({
    mutationFn: async (withPreferences: boolean) => {
      const saved = await profileApi.update(toInput());
      if (withPreferences) {
        const prefs = await profileApi.updatePreferences({
          interestedIn: draft.interestedIn,
          minAge: saved.preferences?.minAge ?? 18,
          maxAge: saved.preferences?.maxAge ?? 99,
          maxDistanceKm: saved.preferences?.maxDistanceKm ?? 50
        });
        return { ...saved, preferences: prefs };
      }
      return saved;
    },
    onSuccess: (saved) => setMyProfile(saved)
  });

  const locate = useMutation({
    mutationFn: async () => {
      const loc = await readApproximateLocation();
      await profileApi.updateLocation(loc.latitude, loc.longitude, loc.city);
      return loc;
    },
    onSuccess: (loc) => setLocationNote(loc.city ? `Position enregistrée : ${loc.city} (environ 1 km).` : "Position approximative enregistrée.")
  });

  const errors: Partial<Record<Step, string | null>> = {
    name: isValidFirstName(draft.firstName) ? null : "Indiquez votre prénom (lettres uniquement).",
    birth: "error" in birth ? birth.error : null,
    gender: draft.gender ? null : "Choisissez une option.",
    seeking: draft.interestedIn.length ? null : "Choisissez au moins une option.",
    photos: photos.length ? null : "Ajoutez au moins une photo pour continuer."
  };
  const currentError = errors[step] ?? null;

  const next = async () => {
    setTouched(true);
    if (currentError) return;
    try {
      if (step === "seeking") await saveProfile.mutateAsync(true);
      if (step === "about") await saveProfile.mutateAsync(false);
    } catch {
      return;
    }
    setTouched(false);
    if (index === STEPS.length - 1) {
      await qc.invalidateQueries({ queryKey: keys.myProfile });
      onFinished();
      return;
    }
    setStep(STEPS[index + 1]);
  };

  const back = () => {
    setTouched(false);
    if (index > 0) setStep(STEPS[index - 1]);
  };

  const age = "iso" in birth ? Math.floor((Date.now() - new Date(birth.iso).getTime()) / 31_557_600_000) : null;

  return (
    <Screen
      footer={
        <View style={styles.footer}>
          <InlineError message={saveProfile.isError ? errorMessage(saveProfile.error) : null} />
          <Button
            label={step === "location" ? "Commencer à découvrir" : "Continuer"}
            loading={saveProfile.isPending}
            onPress={next}
            size="lg"
            testID="onboarding-next"
          />
        </View>
      }
      scroll
    >
      <View style={styles.top}>
        {index > 0 ? <IconButton icon="chevron-back" label="Étape précédente" onPress={back} /> : <IconButton icon="log-out-outline" label="Se déconnecter" onPress={signOut} />}
        <View accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length}`} style={styles.progress}>
          {STEPS.map((s, i) => (
            <View key={s} style={[styles.dot, { backgroundColor: i <= index ? colors.primary : colors.border }]} />
          ))}
        </View>
        <View style={{ width: 44 }} />
      </View>

      {step === "name" && (
        <StepBody subtitle="C'est ainsi qu'il apparaîtra sur votre profil." title="Comment vous appelez-vous ?">
          <TextField
            autoCapitalize="words"
            autoComplete="given-name"
            autoFocus
            error={touched ? currentError : null}
            label="Prénom"
            maxLength={40}
            onChangeText={(firstName) => update({ firstName })}
            onSubmitEditing={next}
            testID="onboarding-firstname"
            value={draft.firstName}
          />
        </StepBody>
      )}

      {step === "birth" && (
        <StepBody subtitle="Seul votre âge sera visible. Lueur est réservée aux 18 ans et plus." title="Votre date de naissance">
          <BirthdateField error={touched ? currentError : null} onChange={(b) => update({ birth: b })} value={draft.birth} />
          {age !== null ? <Text tone="muted">Vous avez {age} ans.</Text> : null}
        </StepBody>
      )}

      {step === "gender" && (
        <StepBody title="Vous êtes…">
          <View style={styles.options}>
            {genders.map((g) => (
              <Chip key={g} label={genderLabels[g]} onPress={() => update({ gender: g })} selected={draft.gender === g} />
            ))}
          </View>
          {touched && currentError ? <Text tone="danger" variant="caption">{currentError}</Text> : null}
        </StepBody>
      )}

      {step === "seeking" && (
        <StepBody subtitle="Vous pourrez changer d'avis à tout moment." title="Qui aimeriez-vous rencontrer ?">
          <View style={styles.options}>
            {genders.map((g) => (
              <Chip
                key={g}
                label={interestedInLabels[g]}
                onPress={() =>
                  update({ interestedIn: draft.interestedIn.includes(g) ? draft.interestedIn.filter((x) => x !== g) : [...draft.interestedIn, g] })
                }
                selected={draft.interestedIn.includes(g)}
              />
            ))}
          </View>
          {touched && currentError ? <Text tone="danger" variant="caption">{currentError}</Text> : null}
          <Text style={styles.subhead} variant="heading">
            Vous cherchez plutôt…
          </Text>
          <View style={styles.options}>
            {lookingFors.map((l) => (
              <Chip key={l} label={lookingForLabels[l]} onPress={() => update({ lookingFor: l })} selected={draft.lookingFor === l} />
            ))}
          </View>
        </StepBody>
      )}

      {step === "photos" && (
        <StepBody subtitle="Une photo récente où l'on voit bien votre visage fait toute la différence." title="Vos photos">
          <PhotoGrid photos={photos} />
          {touched && currentError ? <Text tone="danger" variant="caption">{currentError}</Text> : null}
        </StepBody>
      )}

      {step === "about" && (
        <StepBody subtitle="Facultatif, mais les profils complets reçoivent bien plus de matchs." title="Parlez un peu de vous">
          <TextField
            hint={`${draft.bio.length}/500`}
            label="Bio"
            maxLength={500}
            multiline
            onChangeText={(bio) => update({ bio })}
            placeholder="Ce qui vous fait vibrer, votre dimanche idéal…"
            testID="onboarding-bio"
            value={draft.bio}
          />
          <TextField label="Profession (facultatif)" maxLength={80} onChangeText={(jobTitle) => update({ jobTitle })} value={draft.jobTitle} />
          <Text style={styles.subhead} variant="heading">
            Centres d’intérêt
          </Text>
          <InterestPicker onChange={(interests) => update({ interests })} value={draft.interests} />
        </StepBody>
      )}

      {step === "location" && (
        <StepBody
          subtitle="Pour vous proposer des personnes proches, Lueur utilise une position approximative (environ 1 km). Les autres membres ne voient qu'une distance arrondie, jamais votre position."
          title="Rencontrer près de chez vous"
        >
          <View style={[styles.locationCard, { backgroundColor: colors.accentSoft }]}>
            <Icon color={colors.accent} name="location-outline" size={26} />
            <Text style={styles.flex} tone="muted">
              {locationNote ?? "Vous pouvez aussi passer cette étape : vous verrez alors des profils sans filtre de distance."}
            </Text>
          </View>
          <InlineError message={locate.isError ? errorMessage(locate.error) : null} />
          <Button
            icon="navigate-outline"
            label={locationNote ? "Mettre à jour ma position" : "Utiliser ma position"}
            loading={locate.isPending}
            onPress={() => locate.mutate()}
            testID="onboarding-locate"
            variant="secondary"
          />
        </StepBody>
      )}
    </Screen>
  );
}

function StepBody({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
  return (
    <View style={styles.body}>
      <Text accessibilityRole="header" variant="title">
        {title}
      </Text>
      {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      <View style={styles.content}>{children}</View>
    </View>
  );
}

const styles = StyleSheet.create({
  top: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.lg },
  progress: { flexDirection: "row", gap: 6 },
  dot: { width: 22, height: 5, borderRadius: radii.pill },
  body: { gap: spacing.xs },
  content: { marginTop: spacing.lg, gap: spacing.md },
  options: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  subhead: { marginTop: spacing.sm },
  footer: { gap: spacing.xs },
  locationCard: { flexDirection: "row", gap: spacing.sm, padding: spacing.md, borderRadius: radii.lg, alignItems: "center" },
  flex: { flex: 1 }
});
