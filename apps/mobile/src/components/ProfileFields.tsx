import React from "react";
import { StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { fetchInterests } from "../api/profile";
import { keys } from "../hooks/queries";
import { GENDERS, LOOKING_FOR } from "../lib/labels";
import { MAX_BIO, MAX_INTERESTS, toggleInterest, type DraftErrors, type ProfileDraft } from "../lib/profileDraft";
import { Button, Chip, FormField, Input, Text } from "../shared/ui";
import { spacing } from "../theme";

type Props = {
  draft: ProfileDraft;
  errors: DraftErrors;
  onChange: (patch: Partial<ProfileDraft>) => void;
  onUseLocation: () => void;
  locating?: boolean;
  /** Which part of the form to render (onboarding splits it in two steps). */
  part?: "about" | "more" | "all";
};

export function ProfileFields({ draft, errors, onChange, onUseLocation, locating, part = "all" }: Props) {
  const interests = useQuery({ queryKey: keys.interests, queryFn: fetchInterests, staleTime: Infinity });
  const showAbout = part !== "more";
  const showMore = part !== "about";

  return (
    <View style={styles.form}>
      {showAbout ? (
        <>
          <FormField label="Prénom" error={errors.firstName}>
            <Input
              testID="profile-first-name"
              value={draft.firstName}
              onChangeText={(firstName) => onChange({ firstName })}
              placeholder="Votre prénom"
              maxLength={40}
              autoCapitalize="words"
              invalid={!!errors.firstName}
            />
          </FormField>

          <FormField label="Je suis" error={errors.gender}>
            <View style={styles.chips}>
              {GENDERS.map((g) => (
                <Chip
                  key={g.value}
                  testID={`gender-${g.value}`}
                  label={g.label}
                  selected={draft.gender === g.value}
                  onPress={() => onChange({ gender: g.value })}
                />
              ))}
            </View>
          </FormField>

          <FormField label="À propos de moi" error={errors.bio} hint={`${[...draft.bio].length}/${MAX_BIO}`}>
            <Input
              testID="profile-bio"
              value={draft.bio}
              onChangeText={(bio) => onChange({ bio })}
              placeholder="Quelques mots pour vous présenter…"
              multiline
              maxLength={MAX_BIO}
              invalid={!!errors.bio}
            />
          </FormField>

          <FormField
            label="Ville"
            error={errors.city}
            hint={
              draft.latitude !== undefined && !draft.clearLocation
                ? "Position approximative enregistrée (à ~1 km près)."
                : "Facultatif — sert à vous proposer des personnes proches."
            }
          >
            <Input
              testID="profile-city"
              value={draft.city}
              onChangeText={(city) => onChange({ city })}
              placeholder="Ex. Lyon"
              maxLength={80}
            />
            <Button
              testID="profile-use-location"
              variant="secondary"
              label={locating ? "Localisation…" : "Utiliser ma position approximative"}
              loading={locating}
              onPress={onUseLocation}
            />
          </FormField>
        </>
      ) : null}

      {showMore ? (
        <>
          <FormField label="Je cherche" error={undefined}>
            <View style={styles.chips}>
              {LOOKING_FOR.map((l) => (
                <Chip
                  key={l.value}
                  label={l.label}
                  selected={draft.lookingFor === l.value}
                  onPress={() => onChange({ lookingFor: l.value })}
                />
              ))}
            </View>
          </FormField>

          <FormField label="Centres d'intérêt" error={errors.interests} hint={`${draft.interests.length}/${MAX_INTERESTS} sélectionnés`}>
            <View style={styles.chips}>
              {(interests.data ?? []).map((i) => (
                <Chip
                  key={i.slug}
                  testID={`interest-${i.slug}`}
                  label={i.label}
                  selected={draft.interests.includes(i.slug)}
                  onPress={() => onChange({ interests: toggleInterest(draft.interests, i.slug) })}
                />
              ))}
            </View>
            {interests.isError ? (
              <Text tone="danger" variant="caption">
                Impossible de charger les centres d'intérêt.
              </Text>
            ) : null}
          </FormField>

          <FormField label="Métier" error={errors.occupation}>
            <Input
              value={draft.occupation}
              onChangeText={(occupation) => onChange({ occupation })}
              placeholder="Facultatif"
              maxLength={80}
            />
          </FormField>

          <FormField label="Taille (cm)" error={errors.height}>
            <Input
              value={draft.height}
              onChangeText={(height) => onChange({ height: height.replace(/[^0-9]/g, "") })}
              placeholder="Facultatif"
              keyboardType="number-pad"
              maxLength={3}
              invalid={!!errors.height}
            />
          </FormField>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
