import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { saveLocation, savePreferences, saveProfile } from "../../api/profile";
import { fetchBlocks, unblockUser } from "../../api/social";
import { queryKeys } from "../../api/keys";
import type { Gender, Preferences } from "../../api/types";
import { spacing, useTheme } from "../../theme";
import {
  Avatar,
  Button,
  EmptyState,
  ErrorState,
  Input,
  LoadingState,
  Screen,
  Text,
  useToast
} from "../../ui";
import {
  toBirthDate,
  validateAgeRange,
  validateBio,
  validateBirthDate,
  validateFirstName
} from "../../lib/validation";
import { detectLocation, geocodeCity } from "./location";
import { invalidateProfileData, useInterests, useMyProfile } from "./hooks";
import { PhotoManager } from "./PhotoManager";
import { BioField, GenderPicker, InterestPicker, InterestedInPicker, RangeFields } from "./forms";

export function EditProfileScreen({ navigation }: { navigation: { goBack: () => void } }) {
  const qc = useQueryClient();
  const toast = useToast();
  const query = useMyProfile();
  const interestsQuery = useInterests();
  const [firstName, setFirstName] = useState("");
  const [birth, setBirth] = useState({ day: "", month: "", year: "" });
  const [gender, setGender] = useState<Gender | null>(null);
  const [bio, setBio] = useState("");
  const [city, setCity] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [locating, setLocating] = useState(false);

  const profile = query.data?.profile;
  useEffect(() => {
    if (!profile) return;
    setFirstName(profile.firstName);
    setGender(profile.gender);
    setBio(profile.bio);
    setCity(profile.city);
    setInterests(profile.interests.map((i) => i.slug));
    if (profile.birthDate) {
      const [year, month, day] = profile.birthDate.split("-");
      setBirth({ year, month: String(Number(month)), day: String(Number(day)) });
    }
  }, [profile]);

  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  const updateLocation = async () => {
    setLocating(true);
    try {
      const found = await detectLocation();
      await saveLocation(found.latitude, found.longitude);
      if (found.city) setCity(found.city);
      toast.show("Location updated");
    } catch (e) {
      toast.show(e instanceof Error ? e.message : "Location unavailable.", "error");
    } finally {
      setLocating(false);
    }
  };

  const submit = async () => {
    const birthDate = toBirthDate(birth.year, birth.month, birth.day);
    const problem =
      validateFirstName(firstName) ??
      validateBirthDate(birthDate) ??
      (gender ? null : "Choose your gender.") ??
      validateBio(bio) ??
      (city.trim() ? null : "Tell us your city.");
    setError(problem);
    if (problem || !birthDate || !gender) return;
    setBusy(true);
    try {
      if (profile && city.trim() !== profile.city) {
        const position = await geocodeCity(city.trim());
        await saveLocation(position.latitude, position.longitude);
      }
      await saveProfile({
        firstName: firstName.trim(),
        birthDate,
        gender,
        bio: bio.trim(),
        city: city.trim(),
        interests
      });
      await invalidateProfileData(qc);
      toast.show("Profile saved");
      navigation.goBack();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not save your profile.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll edges={["bottom"]}>
      <Input label="First name" value={firstName} onChangeText={setFirstName} autoCapitalize="words" />
      <View style={styles.dob}>
        <View style={styles.flex}>
          <Input
            label="Day"
            value={birth.day}
            onChangeText={(day) => setBirth({ ...birth, day })}
            keyboardType="number-pad"
            maxLength={2}
          />
        </View>
        <View style={{ width: spacing.sm }} />
        <View style={styles.flex}>
          <Input
            label="Month"
            value={birth.month}
            onChangeText={(month) => setBirth({ ...birth, month })}
            keyboardType="number-pad"
            maxLength={2}
          />
        </View>
        <View style={{ width: spacing.sm }} />
        <View style={styles.flex}>
          <Input
            label="Year"
            value={birth.year}
            onChangeText={(year) => setBirth({ ...birth, year })}
            keyboardType="number-pad"
            maxLength={4}
          />
        </View>
      </View>
      <GenderPicker label="I am" value={gender} onChange={setGender} />
      <Input label="City" value={city} onChangeText={setCity} autoCapitalize="words" />
      <Button
        label="Update my location"
        variant="secondary"
        onPress={() => void updateLocation()}
        loading={locating}
      />
      <View style={{ height: spacing.lg }} />
      <BioField value={bio} onChange={setBio} />
      <InterestPicker
        options={interestsQuery.data?.interests ?? []}
        selected={interests}
        onChange={setInterests}
      />
      {error ? (
        <Text tone="danger" accessibilityRole="alert" style={{ marginBottom: spacing.md }}>
          {error}
        </Text>
      ) : null}
      <Button label="Save" onPress={() => void submit()} loading={busy} />
    </Screen>
  );
}

export function PhotosScreen() {
  const query = useMyProfile();
  const invalidate = async () => {
    await query.refetch();
  };
  return (
    <Screen scroll edges={["bottom"]}>
      {query.isLoading ? (
        <LoadingState />
      ) : (
        <PhotoManager photos={query.data?.profile?.photos ?? []} onChanged={invalidate} />
      )}
    </Screen>
  );
}

function ToggleRow({
  label,
  hint,
  value,
  onChange
}: {
  label: string;
  hint: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <View style={styles.toggle}>
      <View style={styles.flex}>
        <Text variant="label">{label}</Text>
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      </View>
      <Switch value={value} onValueChange={onChange} accessibilityLabel={label} />
    </View>
  );
}

export function PreferencesScreen({ navigation }: { navigation: { goBack: () => void } }) {
  const qc = useQueryClient();
  const toast = useToast();
  const query = useMyProfile();
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const loaded = query.data?.preferences;
  useEffect(() => {
    if (loaded) setPrefs(loaded);
  }, [loaded]);

  const save = useMutation({
    mutationFn: (p: Preferences) => savePreferences(p),
    onSuccess: async () => {
      await invalidateProfileData(qc);
      toast.show("Preferences saved");
      navigation.goBack();
    },
    onError: (e) => toast.show(e instanceof Error ? e.message : "Could not save.", "error")
  });

  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  if (!prefs) return <LoadingState />;
  const problem =
    prefs.interestedIn.length === 0
      ? "Choose at least one option."
      : validateAgeRange(prefs.minAge, prefs.maxAge);

  return (
    <Screen scroll edges={["bottom"]}>
      <InterestedInPicker
        value={prefs.interestedIn}
        onChange={(interestedIn) => setPrefs({ ...prefs, interestedIn })}
      />
      <RangeFields value={prefs} onChange={setPrefs} />
      <ToggleRow
        label="Show me in Discover"
        hint="Turn off to pause your profile without deleting it."
        value={prefs.discoverable}
        onChange={(discoverable) => setPrefs({ ...prefs, discoverable })}
      />
      <ToggleRow
        label="Show my distance"
        hint="Others see a rough distance only. Turn off to hide it."
        value={prefs.showDistance}
        onChange={(showDistance) => setPrefs({ ...prefs, showDistance })}
      />
      {problem ? (
        <Text tone="danger" style={{ marginVertical: spacing.md }}>
          {problem}
        </Text>
      ) : null}
      <Button
        label="Save"
        onPress={() => save.mutate(prefs)}
        loading={save.isPending}
        disabled={problem !== null}
      />
    </Screen>
  );
}

export function BlockedScreen() {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const toast = useToast();
  const query = useQuery({ queryKey: queryKeys.blocks, queryFn: fetchBlocks });
  const unblock = useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.blocks }),
    onError: (e) => toast.show(e instanceof Error ? e.message : "Could not unblock.", "error")
  });

  return (
    <Screen edges={["bottom"]}>
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : (query.data?.blocks.length ?? 0) === 0 ? (
        <EmptyState title="No blocked people" message="People you block will appear here." />
      ) : (
        query.data?.blocks.map((b) => (
          <Pressable
            key={b.userId}
            style={[styles.blockRow, { borderBottomColor: colors.border }]}
            accessibilityRole="none"
          >
            <Avatar url={b.photoUrl} name={b.firstName} size={44} />
            <Text style={styles.flex}>{b.firstName}</Text>
            <Button
              label="Unblock"
              variant="secondary"
              onPress={() => unblock.mutate(b.userId)}
              disabled={unblock.isPending}
            />
          </Pressable>
        ))
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  dob: { flexDirection: "row" },
  toggle: { flexDirection: "row", alignItems: "center", paddingVertical: spacing.md },
  blockRow: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    gap: spacing.md
  }
});
