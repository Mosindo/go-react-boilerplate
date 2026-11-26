import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { updateMyProfile } from "../../api/profile";
import type { Gender } from "../../api/types";
import { useInterests } from "../../hooks/queries";
import { setProfileCache, useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { firstNameProblem } from "../../utils/validation";
import { BirthdateInput, partsFromIso } from "./components/BirthdateInput";
import { GenderPicker, InterestPicker } from "./components/Pickers";

type Props = NativeStackScreenProps<AppStackParamList, "EditProfile">;

export function EditProfileScreen({ navigation }: Props) {
  const { profile } = useAuth();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [gender, setGender] = useState<Gender | "">(profile?.gender ?? "");
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(profile?.jobTitle ?? "");
  const [city, setCity] = useState(profile?.city ?? "");
  const [selected, setSelected] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  if (!profile) return null;

  const save = async () => {
    const problem = firstNameProblem(firstName);
    if (problem) return setError(problem);
    if (!gender) return setError("Choisissez votre genre.");
    setSaving(true);
    setError(null);
    try {
      setProfileCache(
        await updateMyProfile({
          firstName: firstName.trim(),
          birthdate: profile.birthdate,
          gender,
          bio,
          jobTitle,
          city,
          interests: selected
        })
      );
      showToast("Profil enregistré.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen edges={["top", "bottom"]} footer={<Button label="Enregistrer" loading={saving} onPress={save} testID="edit-save" />} scroll>
      <Header onBack={navigation.goBack} title="Modifier le profil" />
      <View style={styles.form}>
        <TextField label="Prénom" onChangeText={setFirstName} value={firstName} />
        <BirthdateInput disabled onChange={() => undefined} value={partsFromIso(profile.birthdate)} />
        <GenderPicker onChange={setGender} value={gender} />
        <TextField label="Bio" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
        <Text style={styles.counter} tone="subtle" variant="caption">
          {bio.length}/500
        </Text>
        <TextField label="Métier" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
        <TextField hint="Un quartier ou une ville suffit." label="Ville" maxLength={60} onChangeText={setCity} value={city} />
        {interests.data ? <InterestPicker catalog={interests.data} onChange={setSelected} value={selected} /> : null}
        {error ? <Text tone="danger">{error}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg, paddingTop: spacing.xs },
  counter: { alignSelf: "flex-end", marginTop: -spacing.md }
});
