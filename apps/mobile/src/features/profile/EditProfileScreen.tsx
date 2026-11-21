import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, RelationshipGoal } from "../../api/types";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { Button, Chip, Header, LoadingState, Screen, Section, showToast, Text, TextField } from "../../ui";
import { ageFromBirthdate, genderLabels, goalLabels } from "../../utils/format";
import { BirthdateField } from "./BirthdateField";
import { useMyProfile, useSetMyProfile } from "./hooks";
import { InterestPicker } from "./InterestPicker";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;
const genders: Gender[] = ["woman", "man", "nonbinary"];
const goals = Object.keys(goalLabels) as RelationshipGoal[];

export function EditProfileScreen({ navigation }: Props) {
  const me = useMyProfile().data;
  if (!me?.profile) return <LoadingState />;
  return <EditProfileForm me={me} onDone={() => navigation.goBack()} />;
}

function EditProfileForm({ me, onDone }: { me: NonNullable<ReturnType<typeof useMyProfile>["data"]>; onDone: () => void }) {
  const styles = useStyles();
  const setProfile = useSetMyProfile();
  const p = me.profile!;
  const [firstName, setFirstName] = useState(p.firstName);
  const [birthdate, setBirthdate] = useState(p.birthdate);
  const [gender, setGender] = useState<Gender>(p.gender);
  const [bio, setBio] = useState(p.bio);
  const [city, setCity] = useState(p.city);
  const [jobTitle, setJobTitle] = useState(p.jobTitle);
  const [height, setHeight] = useState(p.heightCm ? String(p.heightCm) : "");
  const [goal, setGoal] = useState<RelationshipGoal | null>(p.relationshipGoal);
  const [interestIds, setInterestIds] = useState(me.interests.map((i) => i.id));
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const save = async () => {
    const age = ageFromBirthdate(birthdate);
    if (!firstName.trim()) return setError("Indiquez votre prénom.");
    if (age === null || age > 99) return setError("Date de naissance invalide.");
    if (age < 18) return setError("Lueur est réservé aux personnes majeures.");
    const heightCm = height ? Number(height) : null;
    if (heightCm !== null && (!Number.isInteger(heightCm) || heightCm < 120 || heightCm > 230))
      return setError("Taille entre 120 et 230 cm.");
    setSaving(true);
    setError(null);
    try {
      await profileApi.upsert({ firstName: firstName.trim(), birthdate, gender, bio, city, jobTitle, heightCm, relationshipGoal: goal });
      setProfile(await profileApi.interests(interestIds));
      showToast("Profil mis à jour.", "success");
      onDone();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen
      scroll
      keyboard
      footer={
        <>
          {error ? (
            <Text variant="caption" tone="danger" align="center">
              {error}
            </Text>
          ) : null}
          <Button testID="edit-profile-save" label="Enregistrer" size="lg" loading={saving} onPress={() => void save()} />
        </>
      }
    >
      <Header title="Modifier mon profil" />
      <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} />
      <BirthdateField value={birthdate} onChange={setBirthdate} />
      <Section title="Je suis">
        <View style={styles.chips}>
          {genders.map((g) => (
            <Chip key={g} label={genderLabels[g]} selected={gender === g} onPress={() => setGender(g)} />
          ))}
        </View>
      </Section>
      <TextField testID="edit-bio" label="Bio" value={bio} onChangeText={setBio} multiline counter={500} />
      <TextField label="Ville" value={city} onChangeText={setCity} maxLength={80} placeholder="Ex. Lyon" />
      <TextField label="Métier" value={jobTitle} onChangeText={setJobTitle} maxLength={80} />
      <TextField
        label="Taille (cm)"
        value={height}
        onChangeText={(v) => setHeight(v.replace(/\D/g, ""))}
        keyboardType="number-pad"
        maxLength={3}
      />
      <Section title="Je recherche">
        <View style={styles.chips}>
          {goals.map((g) => (
            <Chip key={g} label={goalLabels[g]} selected={goal === g} onPress={() => setGoal(goal === g ? null : g)} />
          ))}
        </View>
      </Section>
      <Section title="Centres d'intérêt">
        <InterestPicker selected={interestIds} onChange={setInterestIds} />
      </Section>
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.sm }
}));
