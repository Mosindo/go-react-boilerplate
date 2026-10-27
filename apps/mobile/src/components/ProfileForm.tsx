import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { Gender, MyProfile, ProfileInput } from "../api/types";
import { formatBirthDateInput, GENDER_LABELS } from "../lib/format";
import { validateProfileDraft, type ProfileErrors } from "../lib/validation";
import { keys } from "../state/keys";
import { Button } from "../ui/Button";
import { Chip } from "../ui/Chip";
import { Input } from "../ui/Input";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";

type Props = {
  initial?: MyProfile | null;
  submitLabel: string;
  saving: boolean;
  onSubmit: (input: ProfileInput) => void;
};

const MAX_INTERESTS = 10;

export function ProfileForm({ initial, submitLabel, saving, onSubmit }: Props) {
  const toast = useToast();
  const interestsQuery = useQuery({
    queryKey: keys.interests,
    queryFn: profileApi.interests,
    staleTime: Infinity
  });
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birthDate, setBirthDate] = useState(initial?.birthDate ?? "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [city, setCity] = useState(initial?.city ?? "");
  const [interests, setInterests] = useState<string[]>(initial?.interests ?? []);
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [locating, setLocating] = useState(false);
  const [errors, setErrors] = useState<ProfileErrors>({});

  const useMyLocation = async () => {
    setLocating(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        toast.show("Location permission denied. You can type your city instead.", "error");
        return;
      }
      const position = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced
      });
      setCoords({ latitude: position.coords.latitude, longitude: position.coords.longitude });
      const places = await Location.reverseGeocodeAsync(position.coords).catch(() => []);
      const name = places[0]?.city ?? places[0]?.subregion ?? places[0]?.region;
      if (name) setCity(name);
      toast.show("Location updated. Only an approximate distance is ever shown.", "success");
    } catch {
      toast.show("Could not get your location.", "error");
    } finally {
      setLocating(false);
    }
  };

  const toggleInterest = (slug: string) =>
    setInterests((current) =>
      current.includes(slug)
        ? current.filter((s) => s !== slug)
        : current.length >= MAX_INTERESTS
          ? current
          : [...current, slug]
    );

  const submit = () => {
    const found = validateProfileDraft({ firstName, birthDate, gender, bio, city });
    setErrors(found);
    if (Object.keys(found).length > 0 || !gender) return;
    onSubmit({
      firstName: firstName.trim(),
      birthDate,
      gender,
      bio: bio.trim(),
      city: city.trim(),
      interests,
      ...(coords ?? {})
    });
  };

  return (
    <View style={styles.form}>
      <Input
        label="First name"
        value={firstName}
        onChangeText={setFirstName}
        error={errors.firstName}
        autoCapitalize="words"
        maxLength={40}
        testID="profile-first-name"
      />
      <Input
        label="Birth date"
        value={birthDate}
        onChangeText={(v) => setBirthDate(formatBirthDateInput(v))}
        error={errors.birthDate}
        placeholder="YYYY-MM-DD"
        keyboardType="number-pad"
        maxLength={10}
        hint="Only your age is shown to others."
        testID="profile-birth-date"
      />
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          I am
        </Text>
        <View style={styles.row}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip
              key={g}
              label={GENDER_LABELS[g]}
              selected={gender === g}
              onPress={() => setGender(g)}
            />
          ))}
        </View>
        {errors.gender ? (
          <Text variant="caption" tone="danger">
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <Input
        label="About me"
        value={bio}
        onChangeText={setBio}
        error={errors.bio}
        multiline
        maxLength={500}
        placeholder="A few words about you"
        hint={`${bio.length}/500`}
      />
      <Input label="City" value={city} onChangeText={setCity} error={errors.city} maxLength={80} />
      <Button
        label={coords || initial?.hasLocation ? "Update my location" : "Use my location"}
        variant="secondary"
        onPress={useMyLocation}
        loading={locating}
      />
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Interests ({interests.length}/{MAX_INTERESTS})
        </Text>
        <View style={[styles.row, styles.wrap]}>
          {(interestsQuery.data ?? []).map((i) => (
            <Chip
              key={i.slug}
              label={i.label}
              selected={interests.includes(i.slug)}
              onPress={() => toggleInterest(i.slug)}
            />
          ))}
        </View>
      </View>
      <Button label={submitLabel} onPress={submit} loading={saving} testID="profile-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 16 },
  group: { gap: 8 },
  row: { flexDirection: "row", gap: 8 },
  wrap: { flexWrap: "wrap" }
});
