import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Interest, OwnProfile, ProfileInput } from "../api/types";
import { parseBirthDate, splitIsoDate } from "../lib/dates";
import { GENDER_LABELS } from "../lib/format";
import { Input, Text, spacing } from "../shared/ui";
import { Chip } from "./Chip";

export const MAX_INTERESTS = 10;

export type ProfileDraft = {
  firstName: string;
  day: string;
  month: string;
  year: string;
  gender: Gender | null;
  bio: string;
  city: string;
  interests: string[];
};

export const emptyDraft: ProfileDraft = { firstName: "", day: "", month: "", year: "", gender: null, bio: "", city: "", interests: [] };

export function draftFromProfile(profile: OwnProfile): ProfileDraft {
  return {
    firstName: profile.firstName,
    ...splitIsoDate(profile.birthDate),
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    interests: profile.interests.map((i) => i.slug)
  };
}

export type DraftErrors = Partial<Record<"firstName" | "birthDate" | "gender" | "bio", string>>;

/** Mirrors the server rules for instant feedback; the server remains the authority. */
export function validateDraft(draft: ProfileDraft, now: Date = new Date()): { errors: DraftErrors; input: ProfileInput | null } {
  const errors: DraftErrors = {};
  const name = draft.firstName.trim();
  if (!name) {
    errors.firstName = "Tell us your first name.";
  } else if (name.length > 40) {
    errors.firstName = "Keep it under 40 characters.";
  }
  const birth = parseBirthDate(draft.day.trim(), draft.month.trim(), draft.year.trim(), now);
  if (!birth.ok) {
    errors.birthDate = birth.error;
  }
  if (!draft.gender) {
    errors.gender = "Pick the option that fits you best.";
  }
  if (draft.bio.trim().length > 500) {
    errors.bio = "Your bio can have up to 500 characters.";
  }
  if (Object.keys(errors).length > 0 || !birth.ok || !draft.gender) {
    return { errors, input: null };
  }
  return {
    errors,
    input: {
      firstName: name,
      birthDate: birth.iso,
      gender: draft.gender,
      bio: draft.bio.trim(),
      city: draft.city.trim(),
      interests: draft.interests
    }
  };
}

type Props = {
  draft: ProfileDraft;
  errors: DraftErrors;
  onChange: (next: ProfileDraft) => void;
  interests: Interest[];
  /** Hide fields that only make sense later in the onboarding flow. */
  section: "basics" | "story";
};

export function ProfileFields({ draft, errors, interests, onChange, section }: Props) {
  const set = <K extends keyof ProfileDraft>(key: K, value: ProfileDraft[K]) => onChange({ ...draft, [key]: value });

  if (section === "basics") {
    return (
      <View style={styles.stack}>
        <Input
          autoCapitalize="words"
          autoComplete="given-name"
          error={errors.firstName}
          label="First name"
          maxLength={40}
          onChangeText={(v) => set("firstName", v)}
          testID="profile-first-name"
          value={draft.firstName}
        />
        <View style={styles.field}>
          <Text variant="label" weight="semibold">
            Birth date
          </Text>
          <View style={styles.dateRow}>
            <Input
              accessibilityLabel="Day"
              containerStyle={styles.dateSmall}
              keyboardType="number-pad"
              maxLength={2}
              onChangeText={(v) => set("day", v.replace(/\D/g, ""))}
              placeholder="DD"
              testID="profile-day"
              value={draft.day}
            />
            <Input
              accessibilityLabel="Month"
              containerStyle={styles.dateSmall}
              keyboardType="number-pad"
              maxLength={2}
              onChangeText={(v) => set("month", v.replace(/\D/g, ""))}
              placeholder="MM"
              testID="profile-month"
              value={draft.month}
            />
            <Input
              accessibilityLabel="Year"
              containerStyle={styles.dateWide}
              keyboardType="number-pad"
              maxLength={4}
              onChangeText={(v) => set("year", v.replace(/\D/g, ""))}
              placeholder="YYYY"
              testID="profile-year"
              value={draft.year}
            />
          </View>
          {errors.birthDate ? (
            <Text tone="danger" variant="caption">
              {errors.birthDate}
            </Text>
          ) : (
            <Text tone="muted" variant="caption">
              Only your age is shown to others. You must be 18 or older.
            </Text>
          )}
        </View>
        <View style={styles.field}>
          <Text variant="label" weight="semibold">
            I am
          </Text>
          <View style={styles.wrap}>
            {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
              <Chip key={g} label={GENDER_LABELS[g]} onPress={() => set("gender", g)} selected={draft.gender === g} testID={`profile-gender-${g}`} />
            ))}
          </View>
          {errors.gender ? (
            <Text tone="danger" variant="caption">
              {errors.gender}
            </Text>
          ) : null}
        </View>
      </View>
    );
  }

  const toggle = (slug: string) => {
    if (draft.interests.includes(slug)) {
      set("interests", draft.interests.filter((s) => s !== slug));
    } else if (draft.interests.length < MAX_INTERESTS) {
      set("interests", [...draft.interests, slug]);
    }
  };

  return (
    <View style={styles.stack}>
      <Input
        error={errors.bio}
        helperText={`${draft.bio.length}/500`}
        label="About you"
        maxLength={500}
        multiline
        onChangeText={(v) => set("bio", v)}
        placeholder="A few words about what you enjoy"
        testID="profile-bio"
        value={draft.bio}
      />
      <Input
        autoComplete="off"
        label="City (optional)"
        maxLength={80}
        onChangeText={(v) => set("city", v)}
        testID="profile-city"
        value={draft.city}
      />
      <View style={styles.field}>
        <Text variant="label" weight="semibold">
          Interests ({draft.interests.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.wrap}>
          {interests.map((i) => (
            <Chip
              disabled={!draft.interests.includes(i.slug) && draft.interests.length >= MAX_INTERESTS}
              key={i.slug}
              label={i.label}
              onPress={() => toggle(i.slug)}
              selected={draft.interests.includes(i.slug)}
            />
          ))}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  stack: { gap: spacing.lg },
  field: { gap: spacing.sm },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  dateRow: { flexDirection: "row", gap: spacing.sm },
  dateSmall: { flex: 1 },
  dateWide: { flex: 1.6 }
});
