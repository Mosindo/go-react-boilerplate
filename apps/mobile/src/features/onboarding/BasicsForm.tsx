import React, { useState } from "react";
import { View } from "react-native";
import type { Gender, OwnProfile } from "../../api/types";
import { Button, Chip, Input, Text, spacing } from "../../shared/ui";
import { isoToFrench, maskBirthDate, parseBirthDate } from "../../utils/dates";
import { GENDERS } from "../../utils/labels";
import { validateFirstName } from "../../utils/validation";

export type BasicsValue = { firstName: string; birthDate?: string; gender: Gender };

type Props = { profile: OwnProfile | null; submitLabel: string; busy?: boolean; onSubmit: (v: BasicsValue) => void };

export function BasicsForm({ profile, submitLabel, busy, onSubmit }: Props) {
  const exists = Boolean(profile?.firstName);
  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [birth, setBirth] = useState(profile?.birthDate ? isoToFrench(profile.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(profile?.firstName ? profile.gender : null);
  const [errors, setErrors] = useState<{ firstName?: string | null; birth?: string | null; gender?: string | null }>({});

  const submit = () => {
    const parsed = exists ? null : parseBirthDate(birth);
    const next = {
      firstName: validateFirstName(firstName),
      birth: parsed && !parsed.ok ? parsed.error : null,
      gender: gender ? null : "Choisissez une option",
    };
    setErrors(next);
    if (next.firstName || next.birth || next.gender || !gender) return;
    onSubmit({ firstName: firstName.trim().replace(/\s+/g, " "), birthDate: parsed && parsed.ok ? parsed.iso : undefined, gender });
  };

  return (
    <View style={{ gap: spacing.lg }}>
      <Input label="Prénom" value={firstName} onChangeText={setFirstName} error={errors.firstName} maxLength={40} autoComplete="given-name" testID="basics-name" />
      {exists ? (
        <View style={{ gap: spacing.xs }}>
          <Text variant="label" tone="muted">Date de naissance</Text>
          <Text>{birth} ({profile?.age} ans)</Text>
          <Text variant="caption" tone="muted">La date de naissance ne peut pas être modifiée.</Text>
        </View>
      ) : (
        <Input label="Date de naissance" value={birth} onChangeText={(t) => setBirth(maskBirthDate(t))} error={errors.birth} keyboardType="number-pad" placeholder="JJ/MM/AAAA" maxLength={10} hint="Vous devez avoir 18 ans ou plus. Votre âge sera affiché, pas votre date." testID="basics-birth" />
      )}
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" tone="muted">Je suis</Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {GENDERS.map((g) => (
            <Chip key={g.value} label={g.label} selected={gender === g.value} onPress={() => setGender(g.value)} testID={`basics-gender-${g.value}`} />
          ))}
        </View>
        {errors.gender ? <Text variant="caption" tone="danger">{errors.gender}</Text> : null}
      </View>
      <Button title={submitLabel} onPress={submit} loading={busy} testID="basics-submit" />
    </View>
  );
}
