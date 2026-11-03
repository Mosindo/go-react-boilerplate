import React from "react";
import { Text, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api";
import type { Gender } from "../../api/types";
import { DISTANCE_OPTIONS, distanceOptionLabel, formatBirthDateInput } from "../../lib/format";
import { spacing, useTheme } from "../../theme";
import { Button, Chip, ChipRow, TextField } from "../../ui/components";
import { GENDERS, MAX_INTERESTS, toggle, type Draft, type Errors } from "./draft";

type FormProps = { draft: Draft; onChange: (next: Draft) => void; errors: Errors };

function Label({ children }: { children: string }) {
  const theme = useTheme();
  return <Text style={{ color: theme.text, fontSize: 15, fontWeight: "600" }}>{children}</Text>;
}

function FieldError({ message }: { message?: string }) {
  const theme = useTheme();
  return message ? (
    <Text accessibilityRole="alert" style={{ color: theme.danger, fontSize: 13 }}>
      {message}
    </Text>
  ) : null;
}

export function BasicsFields({ draft, onChange, errors }: FormProps) {
  return (
    <View style={{ gap: spacing.lg }}>
      <TextField
        label="Prénom"
        value={draft.firstName}
        onChangeText={(firstName) => onChange({ ...draft, firstName })}
        error={errors.firstName}
        maxLength={40}
        autoCapitalize="words"
        testID="field-firstName"
      />
      <TextField
        label="Date de naissance"
        value={draft.birthDate}
        onChangeText={(value) => onChange({ ...draft, birthDate: formatBirthDateInput(value) })}
        error={errors.birthDate}
        hint="Format AAAA-MM-JJ. Seul votre âge est affiché."
        keyboardType="number-pad"
        placeholder="1992-04-12"
        maxLength={10}
        testID="field-birthDate"
      />
      <View style={{ gap: spacing.sm }}>
        <Label>Je suis</Label>
        <ChipRow>
          {GENDERS.map((g) => (
            <Chip
              key={g.value}
              label={g.label}
              selected={draft.gender === g.value}
              onPress={() => onChange({ ...draft, gender: g.value })}
              testID={`gender-${g.value}`}
            />
          ))}
        </ChipRow>
        <FieldError message={errors.gender} />
      </View>
    </View>
  );
}

export function AboutFields({ draft, onChange, errors }: FormProps) {
  const interests = useQuery({ queryKey: ["interests"], queryFn: api.interests, staleTime: Infinity });
  return (
    <View style={{ gap: spacing.lg }}>
      <TextField
        label="À propos de moi"
        value={draft.bio}
        onChangeText={(bio) => onChange({ ...draft, bio })}
        error={errors.bio}
        multiline
        maxLength={500}
        hint={`${draft.bio.length}/500`}
        testID="field-bio"
      />
      <TextField
        label="Ville"
        value={draft.city}
        onChangeText={(city) => onChange({ ...draft, city })}
        error={errors.city}
        maxLength={80}
        autoCapitalize="words"
        testID="field-city"
      />
      <View style={{ gap: spacing.sm }}>
        <Label>{`Centres d'intérêt (${draft.interests.length}/${MAX_INTERESTS})`}</Label>
        <ChipRow>
          {(interests.data ?? []).map((interest) => {
            const selected = draft.interests.includes(interest.slug);
            return (
              <Chip
                key={interest.slug}
                label={interest.label}
                selected={selected}
                onPress={() => {
                  if (!selected && draft.interests.length >= MAX_INTERESTS) {
                    return;
                  }
                  onChange({ ...draft, interests: toggle(draft.interests, interest.slug) });
                }}
                testID={`interest-${interest.slug}`}
              />
            );
          })}
        </ChipRow>
        <FieldError message={errors.interests} />
      </View>
    </View>
  );
}

function Stepper({
  label,
  value,
  min,
  max,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (next: number) => void;
}) {
  const theme = useTheme();
  return (
    <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
      <Label>{label}</Label>
      <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
        <Button
          title="−"
          variant="secondary"
          onPress={() => onChange(Math.max(min, value - 1))}
          style={{ minHeight: 40, paddingHorizontal: spacing.lg }}
        />
        <Text
          accessibilityLabel={`${label} : ${value}`}
          style={{ color: theme.text, fontSize: 18, fontWeight: "700", minWidth: 32, textAlign: "center" }}
        >
          {value}
        </Text>
        <Button
          title="+"
          variant="secondary"
          onPress={() => onChange(Math.min(max, value + 1))}
          style={{ minHeight: 40, paddingHorizontal: spacing.lg }}
        />
      </View>
    </View>
  );
}

export function PreferenceFields({ draft, onChange, errors }: FormProps) {
  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ gap: spacing.sm }}>
        <Label>Je souhaite rencontrer</Label>
        <ChipRow>
          {GENDERS.map((g) => (
            <Chip
              key={g.value}
              label={g.label}
              selected={draft.interestedIn.includes(g.value as Gender)}
              onPress={() => onChange({ ...draft, interestedIn: toggle(draft.interestedIn, g.value) })}
              testID={`interested-${g.value}`}
            />
          ))}
        </ChipRow>
        <FieldError message={errors.interestedIn} />
      </View>
      <Stepper label="Âge minimum" value={draft.ageMin} min={18} max={draft.ageMax} onChange={(ageMin) => onChange({ ...draft, ageMin })} />
      <Stepper
        label="Âge maximum"
        value={draft.ageMax}
        min={draft.ageMin}
        max={120}
        onChange={(ageMax) => onChange({ ...draft, ageMax })}
      />
      <FieldError message={errors.ageMin} />
      <View style={{ gap: spacing.sm }}>
        <Label>Distance maximale</Label>
        <ChipRow>
          {DISTANCE_OPTIONS.map((km) => (
            <Chip
              key={km}
              label={distanceOptionLabel(km)}
              selected={draft.maxDistanceKm === km}
              onPress={() => onChange({ ...draft, maxDistanceKm: km })}
              testID={`distance-${km}`}
            />
          ))}
        </ChipRow>
      </View>
    </View>
  );
}
