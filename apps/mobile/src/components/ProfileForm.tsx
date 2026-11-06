import React from "react";
import { View } from "react-native";
import type { Gender, Interest } from "../api/types";
import { fr } from "../i18n/fr";
import { CITIES } from "../lib/cities";
import { Button, Chip, Input, Text } from "./ui";
import { spacing } from "../theme";
import { isValidFirstName } from "../lib/validation";
import { parseBirthDate } from "../lib/format";

export const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export function GenderPicker({
  value,
  onChange,
  options = fr.onboarding.genders
}: {
  value: Gender | null;
  onChange: (g: Gender) => void;
  options?: Record<string, string>;
}) {
  return (
    <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
      {GENDERS.map((g) => (
        <Chip
          key={g}
          testID={`gender-${g}`}
          label={options[g] ?? g}
          selected={value === g}
          onPress={() => onChange(g)}
        />
      ))}
    </View>
  );
}

export function InterestPicker({
  all,
  selected,
  onChange,
  max = 10
}: {
  all: Interest[];
  selected: string[];
  onChange: (slugs: string[]) => void;
  max?: number;
}) {
  return (
    <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
      {all.map((i) => {
        const on = selected.includes(i.slug);
        return (
          <Chip
            key={i.slug}
            testID={`interest-${i.slug}`}
            label={i.label}
            selected={on}
            onPress={() => {
              if (on) onChange(selected.filter((s) => s !== i.slug));
              else if (selected.length < max) onChange([...selected, i.slug]);
            }}
          />
        );
      })}
    </View>
  );
}

export function CityPicker({
  onPick
}: {
  onPick: (c: { name: string; latitude: number; longitude: number }) => void;
}) {
  return (
    <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
      {CITIES.map((c) => (
        <Chip key={c.name} testID={`city-${c.name}`} label={c.name} onPress={() => onPick(c)} />
      ))}
    </View>
  );
}

export type BasicsDraft = { firstName: string; birthDate: string; gender: Gender | null };

export function basicsErrors(draft: BasicsDraft): {
  firstName?: string;
  birthDate?: string;
  gender?: string;
} {
  const errors: { firstName?: string; birthDate?: string; gender?: string } = {};
  if (!isValidFirstName(draft.firstName)) errors.firstName = fr.onboarding.invalidName;
  const iso = parseBirthDate(draft.birthDate);
  if (!iso) errors.birthDate = fr.onboarding.invalidBirth;
  else {
    const age = Math.floor((Date.now() - new Date(iso).getTime()) / (365.2425 * 24 * 3600 * 1000));
    if (age < 18) errors.birthDate = fr.onboarding.adultOnly;
    else if (age > 99) errors.birthDate = fr.onboarding.invalidBirth;
  }
  if (!draft.gender) errors.gender = fr.onboarding.gender;
  return errors;
}

export function BasicsFields({
  draft,
  onChange,
  showErrors
}: {
  draft: BasicsDraft;
  onChange: (next: BasicsDraft) => void;
  showErrors: boolean;
}) {
  const errors = showErrors ? basicsErrors(draft) : {};
  return (
    <View style={{ gap: spacing.lg }}>
      <Input
        testID="basics-firstname"
        label={fr.onboarding.firstName}
        value={draft.firstName}
        onChangeText={(firstName) => onChange({ ...draft, firstName })}
        autoComplete="given-name"
        error={errors.firstName}
        maxLength={40}
      />
      <Input
        testID="basics-birthdate"
        label={fr.onboarding.birthDate}
        value={draft.birthDate}
        onChangeText={(birthDate) => onChange({ ...draft, birthDate })}
        keyboardType="numbers-and-punctuation"
        placeholder="12/05/1994"
        error={errors.birthDate}
        maxLength={10}
      />
      <View style={{ gap: spacing.sm }}>
        <Text variant="label">{fr.onboarding.gender}</Text>
        <GenderPicker value={draft.gender} onChange={(gender) => onChange({ ...draft, gender })} />
        {errors.gender ? (
          <Text variant="caption" color="#B3261E">
            {errors.gender}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

export function LocationBlock({
  city,
  hasLocation,
  busy,
  denied,
  onLocate,
  onPickCity
}: {
  city: string;
  hasLocation: boolean;
  busy: boolean;
  denied: boolean;
  onLocate: () => void;
  onPickCity: (c: { name: string; latitude: number; longitude: number }) => void;
}) {
  return (
    <View style={{ gap: spacing.md }}>
      <Button
        testID="location-use"
        title={busy ? fr.onboarding.locating : fr.onboarding.useLocation}
        loading={busy}
        onPress={onLocate}
      />
      {denied ? <Text color="#B3261E">{fr.onboarding.locationDenied}</Text> : null}
      <Text variant="label">{fr.onboarding.orPickCity}</Text>
      <CityPicker onPick={onPickCity} />
      {hasLocation ? (
        <Text testID="location-set" color="#2E7D5B" style={{ fontWeight: "600" }}>
          {fr.onboarding.locationSet(city)}
        </Text>
      ) : null}
    </View>
  );
}
