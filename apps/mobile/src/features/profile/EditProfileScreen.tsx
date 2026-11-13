import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, OwnProfile, RelationshipGoal } from "../../api/types";
import type { RootScreenProps } from "../../navigation/types";
import { useMyProfile, useUpdateProfile } from "../../state/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, ErrorState, Header, LoadingState, Screen, Text, TextField, useToast } from "../../ui";
import { describeError } from "../../utils/errors";
import { genderLabels, goalLabels } from "../../utils/format";
import { InterestPicker } from "./InterestPicker";
import { LocationSection } from "./LocationSection";
import { PhotoManager } from "./PhotoManager";

export default function EditProfileScreen(props: RootScreenProps<"EditProfile">) {
  const profile = useMyProfile();
  if (profile.isLoading) return <LoadingState />;
  if (!profile.data) {
    return (
      <Screen>
        <ErrorState message={describeError(profile.error)} onAction={() => void profile.refetch()} />
      </Screen>
    );
  }
  return <EditForm profile={profile.data} {...props} />;
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  const { spacing } = useTheme();
  return (
    <View style={{ gap: spacing.sm }}>
      <Text variant="heading">{title}</Text>
      {children}
    </View>
  );
}

function EditForm({ profile, navigation }: { profile: OwnProfile } & RootScreenProps<"EditProfile">) {
  const { spacing } = useTheme();
  const toast = useToast();
  const update = useUpdateProfile();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender | null>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [height, setHeight] = useState(profile.heightCm ? String(profile.heightCm) : "");
  const [goal, setGoal] = useState<RelationshipGoal | null>(profile.relationshipGoal);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [heightError, setHeightError] = useState<string | null>(null);

  useEffect(() => {
    setInterests(profile.interests.map((i) => i.slug));
  }, [profile.interests]);

  const save = () => {
    const heightCm = height.trim() ? Number(height) : 0;
    if (height.trim() && (!Number.isInteger(heightCm) || heightCm < 120 || heightCm > 230)) {
      setHeightError("Entre 120 et 230 cm.");
      return;
    }
    setHeightError(null);
    update.mutate(
      {
        firstName: firstName.trim(),
        ...(gender ? { gender } : {}),
        bio: bio.trim(),
        jobTitle: jobTitle.trim(),
        heightCm,
        relationshipGoal: goal ?? "",
        interests
      },
      {
        onSuccess: () => {
          toast.show("Profil enregistré.", "success");
          navigation.goBack();
        }
      }
    );
  };

  return (
    <Screen
      scroll
      testID="edit-profile-screen"
      footer={<Button label="Enregistrer" onPress={save} loading={update.isPending} testID="edit-save" />}
    >
      <Header back title="Modifier mon profil" />
      <View style={{ gap: spacing.xl }}>
        <Section title="Photos">
          <Text variant="label" tone="muted">
            Touchez une photo pour la déplacer, la remplacer ou la supprimer.
          </Text>
          <PhotoManager photos={profile.photos} />
        </Section>
        <Section title="Informations">
          <TextField
            label="Prénom"
            value={firstName}
            onChangeText={setFirstName}
            maxLength={40}
            testID="edit-first-name"
          />
          <Text variant="label" weight="medium" tone="muted">
            Genre
          </Text>
          <View style={styles.chips}>
            {(Object.keys(genderLabels) as Gender[]).map((g) => (
              <Chip key={g} label={genderLabels[g]} selected={gender === g} onPress={() => setGender(g)} />
            ))}
          </View>
          <TextField
            label="Métier"
            value={jobTitle}
            onChangeText={setJobTitle}
            maxLength={60}
            placeholder="Facultatif"
          />
          <TextField
            label="Taille (cm)"
            value={height}
            onChangeText={(v) => setHeight(v.replace(/\D/g, "").slice(0, 3))}
            keyboardType="number-pad"
            placeholder="Facultatif"
            error={heightError}
          />
          <Text variant="caption" tone="subtle">
            Votre date de naissance ne peut pas être modifiée.
          </Text>
        </Section>
        <Section title="Bio">
          <TextField
            value={bio}
            onChangeText={setBio}
            multiline
            maxLength={500}
            hint={`${bio.length}/500`}
            accessibilityLabel="Bio"
            testID="edit-bio"
          />
        </Section>
        <Section title="Ce que vous recherchez">
          <View style={styles.chips}>
            {(Object.keys(goalLabels) as RelationshipGoal[]).map((g) => (
              <Chip
                key={g}
                label={goalLabels[g]}
                selected={goal === g}
                onPress={() => setGoal(goal === g ? null : g)}
              />
            ))}
          </View>
        </Section>
        <Section title="Centres d'intérêt">
          <InterestPicker selected={interests} onChange={setInterests} />
        </Section>
        <Section title="Localisation">
          <LocationSection
            city={profile.city}
            hasLocation={profile.hasLocation}
            onDone={() => toast.show("Position mise à jour.", "success")}
          />
        </Section>
        {update.error ? <Text tone="danger">{describeError(update.error)}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
