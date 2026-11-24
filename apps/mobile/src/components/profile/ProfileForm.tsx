import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Profile, RelationshipGoal } from "../../api/types";
import type { ProfileInput } from "../../api/endpoints";
import { colors, spacing } from "../../theme/tokens";
import { genderLabels, goalLabels } from "../../utils/format";
import { checkBirthdate, isValidFirstName } from "../../utils/validation";
import { AppText } from "../AppText";
import { Chip } from "../Chip";
import { TextField } from "../TextField";

export type ProfileDraft = {
  firstName: string;
  day: string;
  month: string;
  year: string;
  gender: Gender | null;
  relationshipGoal: RelationshipGoal;
  bio: string;
  jobTitle: string;
  height: string;
  city: string;
};

export function draftFromProfile(profile: Profile | null): ProfileDraft {
  const [year = "", month = "", day = ""] = profile?.birthdate.split("-") ?? [];
  return {
    firstName: profile?.firstName ?? "",
    day,
    month,
    year,
    gender: profile?.gender ?? null,
    relationshipGoal: profile?.relationshipGoal ?? "unsure",
    bio: profile?.bio ?? "",
    jobTitle: profile?.jobTitle ?? "",
    height: profile?.heightCm ? String(profile.heightCm) : "",
    city: profile?.city ?? ""
  };
}

export type DraftErrors = Partial<Record<"firstName" | "birthdate" | "gender" | "height", string>>;

export function validateIdentity(draft: ProfileDraft): DraftErrors {
  const errors: DraftErrors = {};
  if (!isValidFirstName(draft.firstName)) errors.firstName = "Ton prénom (lettres uniquement, 40 max).";
  const birth = checkBirthdate(draft.day, draft.month, draft.year);
  if (!birth.ok) errors.birthdate = birth.reason;
  if (!draft.gender) errors.gender = "Choisis une option.";
  return errors;
}

export function validateDraft(draft: ProfileDraft): DraftErrors {
  const errors = validateIdentity(draft);
  if (draft.height) {
    const h = Number(draft.height);
    if (!Number.isInteger(h) || h < 120 || h > 230) errors.height = "Entre 120 et 230 cm.";
  }
  return errors;
}

export function draftToInput(draft: ProfileDraft): ProfileInput {
  const birth = checkBirthdate(draft.day, draft.month, draft.year);
  return {
    firstName: draft.firstName.trim(),
    birthdate: birth.ok ? birth.iso : "",
    gender: draft.gender ?? "woman",
    bio: draft.bio.trim(),
    jobTitle: draft.jobTitle.trim(),
    heightCm: draft.height ? Number(draft.height) : null,
    relationshipGoal: draft.relationshipGoal,
    city: draft.city.trim()
  };
}

type Props = {
  draft: ProfileDraft;
  onChange: (draft: ProfileDraft) => void;
  errors: DraftErrors;
  sections?: ("identity" | "about")[];
};

const genders: Gender[] = ["woman", "man", "nonbinary"];
const goals: RelationshipGoal[] = ["long_term", "short_term", "friendship", "unsure"];

export function ProfileForm({ draft, onChange, errors, sections = ["identity", "about"] }: Props) {
  const set = <K extends keyof ProfileDraft>(key: K, value: ProfileDraft[K]) => onChange({ ...draft, [key]: value });
  const digits = (v: string) => v.replace(/\D/g, "");
  return (
    <View style={styles.form}>
      {sections.includes("identity") ? (
        <>
          <TextField
            label="Prénom"
            value={draft.firstName}
            onChangeText={(v) => set("firstName", v)}
            autoComplete="given-name"
            maxLength={40}
            error={errors.firstName}
            testID="profile-first-name"
          />
          <View style={styles.group}>
            <AppText variant="label" color={colors.textMuted}>
              Date de naissance
            </AppText>
            <View style={styles.dateRow}>
              <View style={styles.dateSmall}>
                <TextField placeholder="JJ" value={draft.day} onChangeText={(v) => set("day", digits(v))} keyboardType="number-pad" maxLength={2} accessibilityLabel="Jour de naissance" testID="profile-birth-day" />
              </View>
              <View style={styles.dateSmall}>
                <TextField placeholder="MM" value={draft.month} onChangeText={(v) => set("month", digits(v))} keyboardType="number-pad" maxLength={2} accessibilityLabel="Mois de naissance" testID="profile-birth-month" />
              </View>
              <View style={styles.dateLarge}>
                <TextField placeholder="AAAA" value={draft.year} onChangeText={(v) => set("year", digits(v))} keyboardType="number-pad" maxLength={4} accessibilityLabel="Année de naissance" testID="profile-birth-year" />
              </View>
            </View>
            {errors.birthdate ? (
              <AppText variant="caption" color={colors.danger}>
                {errors.birthdate}
              </AppText>
            ) : (
              <AppText variant="caption" color={colors.textSubtle}>
                Seul ton âge sera visible.
              </AppText>
            )}
          </View>
          <View style={styles.group}>
            <AppText variant="label" color={colors.textMuted}>
              Je suis
            </AppText>
            <View style={styles.chips}>
              {genders.map((g) => (
                <Chip key={g} label={genderLabels[g]} selected={draft.gender === g} onPress={() => set("gender", g)} testID={`profile-gender-${g}`} />
              ))}
            </View>
            {errors.gender ? (
              <AppText variant="caption" color={colors.danger}>
                {errors.gender}
              </AppText>
            ) : null}
          </View>
        </>
      ) : null}
      {sections.includes("about") ? (
        <>
          <View style={styles.group}>
            <AppText variant="label" color={colors.textMuted}>
              Je cherche
            </AppText>
            <View style={styles.chips}>
              {goals.map((g) => (
                <Chip key={g} label={goalLabels[g]} selected={draft.relationshipGoal === g} onPress={() => set("relationshipGoal", g)} testID={`profile-goal-${g}`} />
              ))}
            </View>
          </View>
          <TextField
            label="Ma bio"
            value={draft.bio}
            onChangeText={(v) => set("bio", v)}
            multiline
            maxLength={500}
            counter={500}
            placeholder="Ce qui te fait vibrer, ton dimanche idéal…"
            testID="profile-bio"
          />
          <TextField label="Métier (facultatif)" value={draft.jobTitle} onChangeText={(v) => set("jobTitle", v)} maxLength={80} testID="profile-job" />
          <View style={styles.dateRow}>
            <View style={styles.flex}>
              <TextField label="Taille en cm (facultatif)" value={draft.height} onChangeText={(v) => set("height", digits(v))} keyboardType="number-pad" maxLength={3} error={errors.height} />
            </View>
            <View style={styles.flex}>
              <TextField label="Ville (facultatif)" value={draft.city} onChangeText={(v) => set("city", v)} maxLength={80} autoComplete="postal-address-locality" />
            </View>
          </View>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  dateRow: { flexDirection: "row", gap: spacing.sm },
  dateSmall: { width: 72 },
  dateLarge: { width: 104 },
  flex: { flex: 1 }
});
