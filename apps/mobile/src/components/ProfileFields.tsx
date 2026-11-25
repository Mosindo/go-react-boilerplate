import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { Chip, Text, TextField } from "../ui";
import { genderLabels, genders, lookingForLabels, lookingForOptions } from "../utils/labels";
import { maskBirthdate } from "../utils/validation";
import { InterestPicker } from "./InterestPicker";
import type { ProfileFormErrors, ProfileFormState } from "./profileForm";

type Props = {
  form: ProfileFormState;
  errors: ProfileFormErrors;
  onChange: (patch: Partial<ProfileFormState>) => void;
  sections: ("basics" | "about")[];
};

export function ProfileFields({ form, errors, onChange, sections }: Props) {
  return (
    <View style={styles.wrap}>
      {sections.includes("basics") ? (
        <>
          <TextField
            label="Prénom"
            value={form.firstName}
            onChangeText={(firstName) => onChange({ firstName })}
            autoComplete="given-name"
            maxLength={40}
            error={errors.firstName}
            hint="C'est ce que les autres verront."
            testID="profile-first-name"
          />
          <TextField
            label="Date de naissance"
            value={form.birthdate}
            onChangeText={(v) => onChange({ birthdate: maskBirthdate(v) })}
            keyboardType="number-pad"
            placeholder="JJ/MM/AAAA"
            maxLength={10}
            error={errors.birthdate}
            hint="Seul votre âge est affiché."
            testID="profile-birthdate"
          />
          <View style={styles.group}>
            <Text variant="label" color="textMuted">
              Je suis
            </Text>
            <View style={styles.chips}>
              {genders.map((g) => (
                <Chip
                  key={g}
                  label={genderLabels[g]}
                  selected={form.gender === g}
                  onPress={() => onChange({ gender: g })}
                  testID={`gender-${g}`}
                />
              ))}
            </View>
            {errors.gender ? (
              <Text variant="caption" color="danger">
                {errors.gender}
              </Text>
            ) : null}
          </View>
        </>
      ) : null}

      {sections.includes("about") ? (
        <>
          <TextField
            label="Bio"
            value={form.bio}
            onChangeText={(bio) => onChange({ bio })}
            multiline
            maxLength={500}
            showCounter
            placeholder="Ce qui vous fait vibrer, ce que vous cherchez…"
            error={errors.bio}
            testID="profile-bio"
          />
          <View style={styles.group}>
            <Text variant="label" color="textMuted">
              Je cherche
            </Text>
            <View style={styles.chips}>
              {lookingForOptions.map((l) => (
                <Chip
                  key={l}
                  label={lookingForLabels[l]}
                  selected={form.lookingFor === l}
                  onPress={() => onChange({ lookingFor: form.lookingFor === l ? null : l })}
                />
              ))}
            </View>
          </View>
          <View style={styles.group}>
            <Text variant="label" color="textMuted">
              Centres d’intérêt
            </Text>
            <InterestPicker value={form.interestIds} onChange={(interestIds) => onChange({ interestIds })} />
          </View>
          <View style={styles.row}>
            <View style={styles.flex}>
              <TextField
                label="Métier (optionnel)"
                value={form.jobTitle}
                onChangeText={(jobTitle) => onChange({ jobTitle })}
                maxLength={60}
              />
            </View>
          </View>
          <View style={styles.row}>
            <View style={styles.flex}>
              <TextField
                label="Ville (optionnel)"
                value={form.city}
                onChangeText={(city) => onChange({ city })}
                maxLength={60}
                hint="Affichée telle quelle."
              />
            </View>
            <View style={styles.small}>
              <TextField
                label="Taille (cm)"
                value={form.heightCm}
                onChangeText={(heightCm) => onChange({ heightCm: heightCm.replace(/\D/g, "") })}
                keyboardType="number-pad"
                maxLength={3}
                error={errors.heightCm}
              />
            </View>
          </View>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.md
  },
  group: {
    gap: spacing.xs
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  },
  row: {
    flexDirection: "row",
    gap: spacing.sm
  },
  flex: {
    flex: 1
  },
  small: {
    width: 120
  }
});
