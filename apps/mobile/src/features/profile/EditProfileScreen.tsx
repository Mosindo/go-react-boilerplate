import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Chip, ErrorState, LoadingState, Screen, Text, TextField } from "../../design/components";
import { errorMessage } from "../../lib/api/client";
import { profileApi } from "../../lib/api/endpoints";
import type { Gender, OwnProfile, RelationshipGoal } from "../../lib/api/types";
import { genderLabels, goalLabels } from "../../lib/format";
import { showToast } from "../../lib/toast";
import type { AppStackParamList } from "../../navigation/types";
import { InterestPicker } from "./InterestPicker";
import { LocationControl } from "./LocationControl";
import { PhotoGrid } from "./PhotoGrid";
import { useProfile, useProfileMutation } from "./hooks";

type Props = NativeStackScreenProps<AppStackParamList, "EditProfile">;

export default function EditProfileScreen(props: Props) {
  const { data, isLoading, error, refetch } = useProfile();
  if (isLoading) return <LoadingState />;
  if (error || !data) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;
  return <EditForm profile={data} {...props} />;
}

function EditForm({ profile, navigation }: Props & { profile: OwnProfile }) {
  const [firstName, setFirstName] = useState(profile.firstName);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [gender, setGender] = useState<Gender | null>(profile.gender);
  const [goal, setGoal] = useState<RelationshipGoal | null>(profile.relationshipGoal);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.id));
  const update = useProfileMutation(profileApi.update);
  const updateInterests = useProfileMutation(profileApi.updateInterests);

  const save = async () => {
    try {
      await update.mutateAsync({ firstName, bio, jobTitle, gender: gender ?? undefined, relationshipGoal: goal ?? "" });
      await updateInterests.mutateAsync(interests);
      showToast("Profil enregistré", "success");
      navigation.goBack();
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  return (
    <Screen
      edges={["bottom"]}
      footer={<Button fullWidth label="Enregistrer" loading={update.isPending || updateInterests.isPending} onPress={save} testID="edit-save" />}
      scroll
    >
      <Section title="Photos" subtitle="Touchez une photo pour la définir comme principale, la remplacer ou la supprimer.">
        <PhotoGrid photos={profile.photos} />
      </Section>
      <Section title="À propos">
        <TextField label="Prénom" maxLength={40} onChangeText={setFirstName} value={firstName} />
        <TextField label="Métier (facultatif)" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
        <TextField label="Bio" maxLength={500} multiline onChangeText={setBio} value={bio} hint={`${bio.length}/500`} />
      </Section>
      <Section title="Genre">
        <View style={styles.chips}>
          {(Object.keys(genderLabels) as Gender[]).map((g) => (
            <Chip key={g} label={genderLabels[g]} onPress={() => setGender(g)} selected={gender === g} />
          ))}
        </View>
      </Section>
      <Section title="Ce que je recherche">
        <View style={styles.chips}>
          {(Object.keys(goalLabels) as RelationshipGoal[]).map((g) => (
            <Chip key={g} label={goalLabels[g]} onPress={() => setGoal(goal === g ? null : g)} selected={goal === g} />
          ))}
        </View>
      </Section>
      <Section title="Centres d'intérêt">
        <InterestPicker onChange={setInterests} selected={interests} />
      </Section>
      <Section title="Localisation">
        <LocationControl profile={profile} />
      </Section>
    </Screen>
  );
}

function Section({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text variant="heading">{title}</Text>
      {subtitle ? (
        <Text tone="muted" variant="caption">
          {subtitle}
        </Text>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 10, marginBottom: 12 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
