import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { GENDER_LABELS, type Gender, type OwnProfile } from "../api/types";
import { errorMessage } from "../lib/errors";
import { isoToDisplay, parseBirthDate } from "../lib/dates";
import { queryKeys } from "../lib/queryKeys";
import { Button, Chip, Input, Text, useToast } from "../ui";
import { useInterests } from "./useInterests";

const MAX_INTERESTS = 10;

type Props = { initial?: OwnProfile | null; submitLabel: string; onSaved: () => void };

/** Formats typed digits as JJ/MM/AAAA. */
function maskDate(raw: string): string {
  const d = raw.replace(/\D/g, "").slice(0, 8);
  if (d.length <= 2) return d;
  if (d.length <= 4) return `${d.slice(0, 2)}/${d.slice(2)}`;
  return `${d.slice(0, 2)}/${d.slice(2, 4)}/${d.slice(4)}`;
}

export function ProfileForm({ initial, submitLabel, onSaved }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState(initial ? isoToDisplay(initial.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [city, setCity] = useState(initial?.city ?? "");
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [selected, setSelected] = useState<string[]>(initial?.interests ?? []);
  const [shareLocation, setShareLocation] = useState(!initial);
  const [submitted, setSubmitted] = useState(false);

  const parsed = parseBirthDate(birth);
  const errors = {
    firstName: firstName.trim().length === 0 ? "Votre prénom est requis." : null,
    birth: !parsed
      ? "Format attendu : JJ/MM/AAAA."
      : parsed.age < 18
        ? "Vous devez avoir au moins 18 ans."
        : parsed.age > 99
          ? "Date de naissance invalide."
          : null,
    gender: gender ? null : "Choisissez une option."
  };
  const valid = !errors.firstName && !errors.birth && !errors.gender;

  const save = useMutation({
    mutationFn: async () => {
      if (!parsed || !gender) throw new Error("invalid form");
      const overview = await profileApi.saveProfile({
        firstName: firstName.trim(),
        birthDate: parsed.iso,
        gender,
        bio: bio.trim(),
        city: city.trim(),
        interests: selected
      });
      if (shareLocation) {
        try {
          const perm = await Location.requestForegroundPermissionsAsync();
          if (perm.granted) {
            const pos = await Location.getCurrentPositionAsync({
              accuracy: Location.Accuracy.Balanced
            });
            return await profileApi.saveLocation(pos.coords.latitude, pos.coords.longitude);
          }
          toast.show("Position non partagée : vous verrez des profils sans filtre de distance.");
        } catch {
          toast.show("Impossible d'obtenir votre position.", "error");
        }
      }
      return overview;
    },
    onSuccess: (overview) => {
      qc.setQueryData(queryKeys.overview, overview);
      void qc.invalidateQueries({ queryKey: queryKeys.feed });
      onSaved();
    },
    onError: (err) => toast.show(errorMessage(err), "error")
  });

  const toggle = (slug: string) =>
    setSelected((cur) =>
      cur.includes(slug)
        ? cur.filter((s) => s !== slug)
        : cur.length >= MAX_INTERESTS
          ? cur
          : [...cur, slug]
    );

  return (
    <View style={styles.form}>
      <Input
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        maxLength={40}
        autoCapitalize="words"
        textContentType="givenName"
        error={submitted ? errors.firstName : null}
      />
      <Input
        label="Date de naissance"
        value={birth}
        onChangeText={(v) => setBirth(maskDate(v))}
        placeholder="JJ/MM/AAAA"
        keyboardType="number-pad"
        maxLength={10}
        hint="Seul votre âge sera visible."
        error={submitted ? errors.birth : null}
      />
      <View style={styles.group}>
        <Text variant="label">Je suis</Text>
        <View style={styles.wrap}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip
              key={g}
              label={GENDER_LABELS[g]}
              selected={gender === g}
              onPress={() => setGender(g)}
            />
          ))}
        </View>
        {submitted && errors.gender ? (
          <Text variant="caption" tone="danger">
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <Input label="Ville" value={city} onChangeText={setCity} maxLength={80} placeholder="Lyon" />
      <Input
        label="À propos de moi"
        value={bio}
        onChangeText={setBio}
        maxLength={500}
        multiline
        placeholder="Quelques mots pour briser la glace…"
        hint={`${bio.length}/500`}
      />
      <View style={styles.group}>
        <Text variant="label">
          Centres d'intérêt ({selected.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.wrap}>
          {(interests.data ?? []).map((i) => (
            <Chip
              key={i.slug}
              label={i.label}
              selected={selected.includes(i.slug)}
              onPress={() => toggle(i.slug)}
            />
          ))}
        </View>
      </View>
      <View style={styles.group}>
        <Text variant="label">Position</Text>
        <Chip
          label={
            shareLocation
              ? "Partager ma position approximative"
              : "Ne pas mettre à jour ma position"
          }
          selected={shareLocation}
          onPress={() => setShareLocation((v) => !v)}
        />
        <Text variant="caption" tone="muted">
          Utilisée pour calculer une distance approximative. Vos coordonnées exactes ne sont ni
          stockées ni affichées.
        </Text>
      </View>
      <Button
        label={submitLabel}
        loading={save.isPending}
        onPress={() => {
          setSubmitted(true);
          if (valid) save.mutate();
        }}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 18 },
  group: { gap: 8 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
