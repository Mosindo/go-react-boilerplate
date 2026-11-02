import React, { useEffect, useState } from "react";
import { Pressable, Text, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { Gender, Interest, Preferences, Profile, ProfileInput } from "../api/types";
import { GENDER_LABELS, GENDER_TARGET_LABELS } from "../api/types";
import { birthDateProblem, clamp, firstNameProblem, isoToDisplayDate, parseBirthDate } from "../lib/validation";
import { spacing, useTheme } from "../theme/theme";
import { AppText, Button, Chip, Loading, TextField } from "./ui";
import { useDeviceLocation } from "./useDeviceLocation";

const GENDERS: Gender[] = ["woman", "man", "non_binary"];

/* ----- Identity ----- */

export type IdentityDraft = { firstName: string; birthDate: string; gender: Gender | null };

export function IdentityFields({
  draft,
  onChange,
  errors
}: {
  draft: IdentityDraft;
  onChange: (d: IdentityDraft) => void;
  errors: Partial<Record<"firstName" | "birthDate" | "gender", string>>;
}) {
  return (
    <View style={{ gap: spacing.md }}>
      <TextField
        label="Prénom"
        value={draft.firstName}
        onChangeText={(firstName) => onChange({ ...draft, firstName })}
        maxLength={40}
        error={errors.firstName}
        autoComplete="given-name"
        testID="field-firstName"
      />
      <TextField
        label="Date de naissance"
        value={draft.birthDate}
        onChangeText={(birthDate) => onChange({ ...draft, birthDate })}
        placeholder="JJ/MM/AAAA"
        keyboardType="numbers-and-punctuation"
        maxLength={10}
        error={errors.birthDate}
        hint="Seul votre âge est affiché."
        testID="field-birthDate"
      />
      <AppText variant="label">Je suis</AppText>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {GENDERS.map((g) => (
          <Chip key={g} label={GENDER_LABELS[g]} selected={draft.gender === g} onPress={() => onChange({ ...draft, gender: g })} />
        ))}
      </View>
      {errors.gender ? (
        <AppText color="#B3261E" variant="caption">
          {errors.gender}
        </AppText>
      ) : null}
    </View>
  );
}

export function validateIdentity(draft: IdentityDraft): Partial<Record<"firstName" | "birthDate" | "gender", string>> {
  const errors: Partial<Record<"firstName" | "birthDate" | "gender", string>> = {};
  const name = firstNameProblem(draft.firstName);
  if (name) errors.firstName = name;
  const birth = birthDateProblem(draft.birthDate);
  if (birth) errors.birthDate = birth;
  if (!draft.gender) errors.gender = "Choisissez une option.";
  return errors;
}

export function identityFromProfile(profile: Profile | null): IdentityDraft {
  return {
    firstName: profile?.firstName ?? "",
    birthDate: profile ? isoToDisplayDate(profile.birthDate) : "",
    gender: profile?.gender ?? null
  };
}

/* ----- Preferences ----- */

export function PreferenceFields({ value, onChange }: { value: Preferences; onChange: (p: Preferences) => void }) {
  const { colors } = useTheme();
  const toggle = (g: Gender) => {
    const has = value.interestedIn.includes(g);
    const next = has ? value.interestedIn.filter((x) => x !== g) : [...value.interestedIn, g];
    if (next.length > 0) onChange({ ...value, interestedIn: next });
  };
  const step = (key: "minAge" | "maxAge" | "maxDistanceKm", delta: number) => {
    const next = { ...value };
    if (key === "minAge") next.minAge = clamp(value.minAge + delta, 18, value.maxAge);
    if (key === "maxAge") next.maxAge = clamp(value.maxAge + delta, value.minAge, 99);
    if (key === "maxDistanceKm") next.maxDistanceKm = clamp(value.maxDistanceKm + delta, 5, 500);
    onChange(next);
  };
  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ gap: spacing.sm }}>
        <AppText variant="label">Je souhaite rencontrer</AppText>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {GENDERS.map((g) => (
            <Chip key={g} label={GENDER_TARGET_LABELS[g]} selected={value.interestedIn.includes(g)} onPress={() => toggle(g)} />
          ))}
        </View>
      </View>
      <Stepper
        label="Âge minimum"
        value={`${value.minAge} ans`}
        onMinus={() => step("minAge", -1)}
        onPlus={() => step("minAge", 1)}
        colors={colors}
      />
      <Stepper
        label="Âge maximum"
        value={`${value.maxAge} ans`}
        onMinus={() => step("maxAge", -1)}
        onPlus={() => step("maxAge", 1)}
        colors={colors}
      />
      <Stepper
        label="Distance maximale"
        value={`${value.maxDistanceKm} km`}
        onMinus={() => step("maxDistanceKm", value.maxDistanceKm > 50 ? -25 : -5)}
        onPlus={() => step("maxDistanceKm", value.maxDistanceKm >= 50 ? 25 : 5)}
        colors={colors}
      />
    </View>
  );
}

function Stepper({
  label,
  value,
  onMinus,
  onPlus,
  colors
}: {
  label: string;
  value: string;
  onMinus: () => void;
  onPlus: () => void;
  colors: ReturnType<typeof useTheme>["colors"];
}) {
  const btn = {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: colors.surfaceAlt,
    alignItems: "center" as const,
    justifyContent: "center" as const
  };
  return (
    <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
      <AppText variant="label">{label}</AppText>
      <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
        <Pressable accessibilityRole="button" accessibilityLabel={`${label} : diminuer`} onPress={onMinus} style={btn}>
          <Text style={{ color: colors.text, fontSize: 22 }}>−</Text>
        </Pressable>
        <Text accessibilityLiveRegion="polite" style={{ color: colors.text, fontWeight: "700", minWidth: 70, textAlign: "center" }}>
          {value}
        </Text>
        <Pressable accessibilityRole="button" accessibilityLabel={`${label} : augmenter`} onPress={onPlus} style={btn}>
          <Text style={{ color: colors.text, fontSize: 22 }}>+</Text>
        </Pressable>
      </View>
    </View>
  );
}

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], minAge: 18, maxAge: 50, maxDistanceKm: 50 };

/* ----- Location ----- */

export type LocationDraft = { latitude: number | null; longitude: number | null; city: string };

export function LocationFields({ value, onChange }: { value: LocationDraft; onChange: (l: LocationDraft) => void }) {
  const { locate, geocode, busy, error } = useDeviceLocation();
  const [query, setQuery] = useState("");
  return (
    <View style={{ gap: spacing.md }}>
      <AppText muted>
        Votre position sert uniquement à trouver des personnes proches. Elle est arrondie à environ 1 km et personne ne voit vos coordonnées
        : seule une distance approximative est affichée.
      </AppText>
      <Button
        label={busy ? "Localisation…" : "Utiliser ma position"}
        onPress={async () => {
          const r = await locate();
          if (r) onChange(r);
        }}
        loading={busy}
        testID="locate-me"
      />
      <TextField
        label="Ou saisir une ville"
        value={query}
        onChangeText={setQuery}
        placeholder="Lyon, Bruxelles, Montréal…"
        returnKeyType="search"
        onSubmitEditing={async () => {
          const r = await geocode(query);
          if (r) onChange(r);
        }}
      />
      <Button
        label="Rechercher cette ville"
        variant="secondary"
        disabled={query.trim().length < 2}
        onPress={async () => {
          const r = await geocode(query);
          if (r) onChange(r);
        }}
      />
      {error ? <AppText color="#B3261E">{error}</AppText> : null}
      {value.latitude !== null ? <AppText color="#2F8F5B">✓ Position enregistrée{value.city ? ` · ${value.city}` : ""}</AppText> : null}
    </View>
  );
}

/* ----- Bio + interests ----- */

export function AboutFields({
  bio,
  onBio,
  selected,
  onSelected
}: {
  bio: string;
  onBio: (b: string) => void;
  selected: number[];
  onSelected: (ids: number[]) => void;
}) {
  const interests = useQuery({ queryKey: ["interests"], queryFn: profileApi.interests, staleTime: Infinity });
  const toggle = (item: Interest) => {
    if (selected.includes(item.id)) onSelected(selected.filter((id) => id !== item.id));
    else if (selected.length < 10) onSelected([...selected, item.id]);
  };
  return (
    <View style={{ gap: spacing.md }}>
      <TextField
        label="À propos de moi"
        value={bio}
        onChangeText={onBio}
        multiline
        maxLength={500}
        style={{ minHeight: 110, textAlignVertical: "top" }}
        hint={`${bio.length}/500`}
        testID="field-bio"
      />
      <AppText variant="label">Centres d'intérêt ({selected.length}/10)</AppText>
      {interests.isLoading ? (
        <Loading />
      ) : (
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {(interests.data?.items ?? []).map((item) => (
            <Chip key={item.id} label={item.label} selected={selected.includes(item.id)} onPress={() => toggle(item)} />
          ))}
        </View>
      )}
    </View>
  );
}

/* ----- helpers ----- */

export function buildProfileInput(
  identity: IdentityDraft,
  location: LocationDraft,
  bio: string,
  interestIds: number[],
  flags: { showDistance: boolean; discoverable: boolean }
): ProfileInput | null {
  const parsed = parseBirthDate(identity.birthDate);
  if (!parsed || !identity.gender) return null;
  return {
    firstName: identity.firstName.trim(),
    birthDate: parsed.iso,
    gender: identity.gender,
    bio: bio.trim(),
    city: location.city.trim(),
    latitude: location.latitude,
    longitude: location.longitude,
    showDistance: flags.showDistance,
    discoverable: flags.discoverable,
    interestIds
  };
}

export function useSyncedState<T>(source: T): [T, React.Dispatch<React.SetStateAction<T>>] {
  const [state, setState] = useState(source);
  useEffect(() => setState(source), [source]);
  return [state, setState];
}
