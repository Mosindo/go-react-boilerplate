import { useQueryClient } from "@tanstack/react-query";
import React, { useMemo, useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { saveProfile } from "../api/profile";
import type { Gender, OwnProfile } from "../api/types";
import { useAsyncAction } from "../hooks/useAsyncAction";
import { bumpDiscoverVersion } from "../state/discoverStore";
import { queryKeys, useInterests } from "../hooks/queries";
import { fromIsoDate, validateBirthDate, type BirthDateParts } from "../lib/date";
import { GENDER_LABELS } from "../lib/format";
import { geocodeCity, locateMe } from "../lib/location";
import { fontSizes, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Chip } from "./Chip";
import { TextField } from "./TextField";

const MAX_INTERESTS = 10;
const MAX_BIO = 500;

type Props = {
  initial: OwnProfile | null;
  submitLabel: string;
  onSaved: (profile: OwnProfile) => void;
};

export function ProfileForm({ initial, submitLabel, onSaved }: Props) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const interestsQuery = useInterests();

  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState<BirthDateParts>(
    initial ? fromIsoDate(initial.birthDate) : { day: "", month: "", year: "" }
  );
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [city, setCity] = useState(initial?.city ?? "");
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(
    initial?.interests.map((i) => i.id) ?? []
  );
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});

  const birthLocked = initial !== null; // changing a declared age after sign-up is not allowed in the UI
  const hasLocation = coords !== null || Boolean(initial?.hasLocation);

  const locate = useAsyncAction(async () => {
    const position = await locateMe();
    setCoords({ latitude: position.latitude, longitude: position.longitude });
    if (position.city) setCity(position.city);
    setErrors((e) => ({ ...e, city: undefined }));
  });

  const geocode = useAsyncAction(async () => {
    const position = await geocodeCity(city);
    setCoords({ latitude: position.latitude, longitude: position.longitude });
    setErrors((e) => ({ ...e, city: undefined }));
  });

  const submit = useAsyncAction(async () => {
    const next: Record<string, string | undefined> = {};
    const name = firstName.trim().replace(/\s+/g, " ");
    if (!name) next.firstName = "Entrez votre prénom.";
    if (name.length > 40) next.firstName = "40 caractères maximum.";

    let birthDate = initial?.birthDate ?? "";
    if (!birthLocked) {
      const result = validateBirthDate(birth);
      if ("error" in result) next.birth = result.error;
      else birthDate = result.iso;
    }
    if (!gender) next.gender = "Choisissez une option.";
    if (!hasLocation)
      next.city = "Indiquez où vous vous trouvez pour voir des profils près de vous.";
    if (bio.length > MAX_BIO) next.bio = `${MAX_BIO} caractères maximum.`;
    setErrors(next);
    if (Object.values(next).some(Boolean) || !gender) {
      return;
    }

    // A city typed but never validated: resolve it now so the stored position matches the label.
    let position = coords;
    const cityChanged = city.trim() !== (initial?.city ?? "");
    if (!position && city.trim() && (cityChanged || !hasLocation)) {
      const found = await geocodeCity(city);
      position = { latitude: found.latitude, longitude: found.longitude };
    }

    const profile = await saveProfile({
      firstName: name,
      birthDate,
      gender,
      bio: bio.trim(),
      city: city.trim(),
      ...(position ?? {}),
      interestIds,
      isVisible: initial?.isVisible,
      showDistance: initial?.showDistance
    });
    queryClient.setQueryData(queryKeys.profile, profile);
    bumpDiscoverVersion();
    onSaved(profile);
  });

  const toggleInterest = (id: number) =>
    setInterestIds((current) =>
      current.includes(id)
        ? current.filter((x) => x !== id)
        : current.length >= MAX_INTERESTS
          ? current
          : [...current, id]
    );

  const birthInput = useMemo(
    () => (
      <View>
        <Text style={[styles.label, { color: theme.textMuted }]}>Date de naissance</Text>
        <View style={styles.birthRow}>
          <View style={styles.birthSmall}>
            <TextField
              editable={!birthLocked}
              keyboardType="number-pad"
              label="Jour"
              maxLength={2}
              onChangeText={(day) => setBirth((b) => ({ ...b, day }))}
              placeholder="JJ"
              value={birth.day}
            />
          </View>
          <View style={styles.birthSmall}>
            <TextField
              editable={!birthLocked}
              keyboardType="number-pad"
              label="Mois"
              maxLength={2}
              onChangeText={(month) => setBirth((b) => ({ ...b, month }))}
              placeholder="MM"
              value={birth.month}
            />
          </View>
          <View style={styles.birthYear}>
            <TextField
              editable={!birthLocked}
              keyboardType="number-pad"
              label="Année"
              maxLength={4}
              onChangeText={(year) => setBirth((b) => ({ ...b, year }))}
              placeholder="AAAA"
              value={birth.year}
            />
          </View>
        </View>
        {errors.birth ? (
          <Text style={[styles.error, { color: theme.danger }]}>{errors.birth}</Text>
        ) : null}
        {birthLocked ? (
          <Text style={[styles.hint, { color: theme.textMuted }]}>
            Votre date de naissance ne peut pas être modifiée.
          </Text>
        ) : null}
      </View>
    ),
    [birth, birthLocked, errors.birth, theme]
  );

  return (
    <View>
      <TextField
        autoCapitalize="words"
        error={errors.firstName}
        label="Prénom"
        maxLength={40}
        onChangeText={setFirstName}
        value={firstName}
      />

      {birthInput}

      <Text style={[styles.label, { color: theme.textMuted }]}>Je suis</Text>
      <View style={styles.chips}>
        {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
          <Chip
            key={g}
            label={GENDER_LABELS[g]}
            onPress={() => setGender(g)}
            selected={gender === g}
          />
        ))}
      </View>
      {errors.gender ? (
        <Text style={[styles.error, { color: theme.danger }]}>{errors.gender}</Text>
      ) : null}

      <View style={styles.spacer} />
      <TextField
        error={errors.city ?? locate.error ?? geocode.error}
        hint="Seule une distance approximative est montrée aux autres membres."
        label="Ville"
        maxLength={80}
        onChangeText={(value) => {
          setCity(value);
          setCoords(null);
        }}
        value={city}
      />
      <View style={styles.row}>
        <Button
          loading={locate.loading}
          onPress={() => void locate.run()}
          style={styles.half}
          title="Utiliser ma position"
          variant="secondary"
        />
        <Button
          disabled={!city.trim()}
          loading={geocode.loading}
          onPress={() => void geocode.run()}
          style={styles.half}
          title="Valider la ville"
          variant="secondary"
        />
      </View>
      <Text style={[styles.hint, { color: hasLocation ? theme.success : theme.textMuted }]}>
        {hasLocation ? "✓ Position enregistrée (approximative)." : "Position non définie."}
      </Text>

      <View style={styles.spacer} />
      <TextField
        error={errors.bio}
        hint={`${bio.length}/${MAX_BIO}`}
        label="À propos de moi"
        multiline
        onChangeText={setBio}
        placeholder="Quelques mots sur vous…"
        value={bio}
      />

      <Text style={[styles.label, { color: theme.textMuted }]}>
        Centres d&apos;intérêt ({interestIds.length}/{MAX_INTERESTS})
      </Text>
      <View style={styles.chips}>
        {(interestsQuery.data ?? []).map((interest) => (
          <Chip
            key={interest.id}
            label={interest.label}
            onPress={() => toggleInterest(interest.id)}
            selected={interestIds.includes(interest.id)}
          />
        ))}
      </View>

      {submit.error ? (
        <Text style={[styles.error, { color: theme.danger }]}>{submit.error}</Text>
      ) : null}
      <Button
        loading={submit.loading}
        onPress={() => void submit.run()}
        style={styles.submit}
        testID="profile-submit"
        title={submitLabel}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  label: {
    fontSize: fontSizes.xs,
    fontWeight: "700",
    letterSpacing: 0.6,
    marginBottom: spacing.sm,
    textTransform: "uppercase"
  },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  birthRow: { flexDirection: "row", gap: spacing.sm },
  birthSmall: { flex: 1 },
  birthYear: { flex: 1.6 },
  row: { flexDirection: "row", gap: spacing.sm },
  half: { flex: 1, minHeight: 44 },
  hint: { fontSize: fontSizes.xs, marginTop: spacing.sm },
  error: { fontSize: fontSizes.xs, marginBottom: spacing.sm },
  spacer: { height: spacing.md },
  submit: { marginTop: spacing.xl }
});
