import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { MeProfile, Photo, RelationshipGoal } from "../../api/types";
import { keys, useMyProfile, useSetMyProfile } from "../../hooks/queries";
import type { MainScreenProps } from "../../navigation/types";
import { Button, Chip, Header, LoadingState, Screen, Text, TextField, showToast } from "../../ui";
import { goalLabels } from "../../utils/format";
import { isValidFirstName } from "../../utils/validation";
import { InterestPicker } from "./InterestPicker";
import { PhotoGrid } from "./PhotoGrid";

const GOALS = Object.keys(goalLabels) as RelationshipGoal[];

export function EditProfileScreen(props: MainScreenProps<"EditProfile">) {
  const { data } = useMyProfile();
  if (!data?.profile)
    return (
      <Screen>
        <Header back title="Modifier" />
        <LoadingState />
      </Screen>
    );
  return <EditForm me={data} {...props} />;
}

function EditForm({ me, navigation }: { me: MeProfile } & MainScreenProps<"EditProfile">) {
  const qc = useQueryClient();
  const setMe = useSetMyProfile();
  const p = me.profile!;
  const [firstName, setFirstName] = useState(p.firstName);
  const [bio, setBio] = useState(p.bio);
  const [jobTitle, setJobTitle] = useState(p.jobTitle);
  const [city, setCity] = useState(p.city);
  const [height, setHeight] = useState(p.heightCm ? String(p.heightCm) : "");
  const [goal, setGoal] = useState<RelationshipGoal | null>(p.relationshipGoal);
  const [interestIds, setInterestIds] = useState(me.interests.map((i) => i.id));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onPhotos = (photos: Photo[]) => setMe((prev) => ({ ...prev, photos }));

  const save = async () => {
    setError(null);
    if (!isValidFirstName(firstName)) return setError("Prénom invalide.");
    const heightCm = height.trim() ? Number(height) : null;
    if (heightCm !== null && (!Number.isInteger(heightCm) || heightCm < 120 || heightCm > 230)) return setError("Taille entre 120 et 230 cm.");
    setSaving(true);
    try {
      const updated = await profileApi.upsert({
        firstName: firstName.trim(),
        birthdate: p.birthdate,
        gender: p.gender,
        bio: bio.trim(),
        jobTitle: jobTitle.trim(),
        city: city.trim(),
        heightCm,
        relationshipGoal: goal,
        interestIds
      });
      qc.setQueryData(keys.me, updated);
      showToast("Profil enregistré.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen
      scroll
      footer={
        <View style={styles.footer}>
          {error ? (
            <Text tone="danger" variant="small">
              {error}
            </Text>
          ) : null}
          <Button label="Enregistrer" loading={saving} onPress={save} testID="edit-save" />
        </View>
      }
    >
      <Header back title="Modifier mon profil" />
      <View style={styles.body}>
        <Text variant="heading">Photos</Text>
        <PhotoGrid onChange={onPhotos} photos={me.photos} />
        <TextField label="Prénom" maxLength={30} onChangeText={setFirstName} value={firstName} />
        <TextField counter label="Bio" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
        <TextField label="Métier" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
        <TextField hint="Affichée sur votre profil, indépendamment de votre position." label="Ville" maxLength={60} onChangeText={setCity} value={city} />
        <TextField
          inputMode="numeric"
          keyboardType="number-pad"
          label="Taille en cm (facultatif)"
          maxLength={3}
          onChangeText={(v) => setHeight(v.replace(/\D/g, ""))}
          value={height}
        />
        <Text variant="heading">Ce que je cherche</Text>
        <View style={styles.chips}>
          {GOALS.map((g) => (
            <Chip key={g} label={goalLabels[g]} onPress={() => setGoal(goal === g ? null : g)} selected={goal === g} />
          ))}
        </View>
        <Text variant="heading">Centres d’intérêt</Text>
        <InterestPicker onChange={setInterestIds} value={interestIds} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { gap: 16 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  footer: { gap: 8 }
});
