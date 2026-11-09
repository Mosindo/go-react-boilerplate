import React, { useState } from "react";
import { Platform, StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import type { Gender, OwnProfile, ProfileInput } from "../api/types";
import { ageOn, formatBirthDateInput, isoToDisplay, parseBirthDate } from "../lib/dates";
import { Button, Chip, Input, Text, useToast } from "../ui";
import { useInterests } from "./hooks";

const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "nonbinary", label: "Non-binaire" }
];

const MAX_INTERESTS = 10;

type Props = {
  initial: OwnProfile | null;
  submitLabel: string;
  busy: boolean;
  onSubmit: (input: ProfileInput) => void;
};

export function ProfileForm({ initial, submitLabel, busy, onSubmit }: Props) {
  const toast = useToast();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState(initial ? isoToDisplay(initial.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [city, setCity] = useState(initial?.city ?? "");
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [selected, setSelected] = useState<number[]>(initial?.interests.map((i) => i.id) ?? []);
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [locating, setLocating] = useState(false);
  const [touched, setTouched] = useState(false);

  const iso = parseBirthDate(birth);
  const age = iso ? ageOn(iso, new Date()) : null;
  const errors = {
    firstName: firstName.trim() ? null : "Votre prénom est requis.",
    birth:
      iso === null
        ? "Format JJ/MM/AAAA."
        : age !== null && age < 18
          ? "Vous devez avoir 18 ans ou plus."
          : age !== null && age > 100
            ? "Date non valide."
            : null,
    gender: gender ? null : "Choisissez une option."
  };
  const valid = !errors.firstName && !errors.birth && !errors.gender;

  const toggleInterest = (id: number) =>
    setSelected((current) => {
      if (current.includes(id)) return current.filter((x) => x !== id);
      if (current.length >= MAX_INTERESTS) {
        toast.show(`${MAX_INTERESTS} centres d'intérêt maximum.`, "error");
        return current;
      }
      return [...current, id];
    });

  const locate = async () => {
    setLocating(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        toast.show("Position refusée. Indiquez votre ville manuellement.", "error");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      setCoords({ latitude: position.coords.latitude, longitude: position.coords.longitude });
      if (!city.trim() && Platform.OS !== "web") {
        const places = await Location.reverseGeocodeAsync(position.coords).catch(() => []);
        const name = places[0]?.city ?? places[0]?.subregion;
        if (name) setCity(name);
      }
      toast.show("Position enregistrée (arrondie, jamais affichée précisément).");
    } catch {
      toast.show("Impossible d'obtenir votre position.", "error");
    } finally {
      setLocating(false);
    }
  };

  const submit = () => {
    setTouched(true);
    if (!valid || !iso || !gender) return;
    onSubmit({
      firstName: firstName.trim(),
      birthDate: iso,
      gender,
      city: city.trim(),
      bio: bio.trim(),
      interestIds: selected,
      isVisible: initial?.isVisible,
      showDistance: initial?.showDistance,
      ...(coords ?? {})
    });
  };

  return (
    <View style={styles.form}>
      <Input
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        maxLength={40}
        autoCapitalize="words"
        error={touched ? errors.firstName : null}
        testID="profile-first-name"
      />
      <Input
        label="Date de naissance"
        value={birth}
        onChangeText={(t) => setBirth(formatBirthDateInput(t))}
        placeholder="JJ/MM/AAAA"
        keyboardType="number-pad"
        maxLength={10}
        error={touched || birth.length === 10 ? errors.birth : null}
        testID="profile-birth"
      />
      <View style={styles.group}>
        <Text variant="label" muted>
          Je suis
        </Text>
        <View style={styles.row}>
          {GENDERS.map((g) => (
            <Chip key={g.value} label={g.label} selected={gender === g.value} onPress={() => setGender(g.value)} />
          ))}
        </View>
        {touched && errors.gender ? (
          <Text variant="caption" color="#B3261E">
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <Input label="Ville" value={city} onChangeText={setCity} maxLength={80} />
      <Button
        title={coords || initial?.hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
        variant="secondary"
        onPress={locate}
        loading={locating}
      />
      <Text variant="caption" muted>
        Votre position sert uniquement à trouver des profils proches. Les autres voient une distance approximative, jamais votre
        emplacement.
      </Text>
      <Input label="À propos de moi" value={bio} onChangeText={setBio} multiline maxLength={500} placeholder="Quelques mots sur vous…" />
      <Text variant="caption" muted style={styles.counter}>
        {bio.length}/500
      </Text>
      <View style={styles.group}>
        <Text variant="label" muted>
          Centres d'intérêt ({selected.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.wrap}>
          {(interests.data ?? []).map((i) => (
            <Chip key={i.id} label={i.label} selected={selected.includes(i.id)} onPress={() => toggleInterest(i.id)} />
          ))}
        </View>
      </View>
      <Button title={submitLabel} onPress={submit} loading={busy} testID="profile-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 16 },
  group: { gap: 8 },
  row: { flexDirection: "row", gap: 8, flexWrap: "wrap" },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  counter: { textAlign: "right", marginTop: -8 }
});
