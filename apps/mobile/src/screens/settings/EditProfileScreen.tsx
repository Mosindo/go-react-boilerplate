import React, { useState } from "react";
import { View } from "react-native";
import { errorMessage } from "../../api/errors";
import type { Gender, ProfileView, RelationshipGoal } from "../../api/types";
import { ChoiceGroup } from "../../components/ChoiceGroup";
import { InterestPicker } from "../../components/InterestPicker";
import { useFeedback } from "../../feedback";
import { useInterestsCatalog, useMyProfile, useSaveInterests, useSaveProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Card, Header, LoadingState, Screen, Text, TextField } from "../../ui";
import { genderLabels, genders, goalLabels, goals } from "../../utils/labels";
import { validateFirstName } from "../../utils/validation";

export default function EditProfileScreen({ navigation }: AppScreenProps<"EditProfile">) {
  const query = useMyProfile();
  if (!query.data?.profile) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  return (
    <EditProfileForm interests={query.data.interests.map((i) => i.slug)} onDone={() => navigation.goBack()} profile={query.data.profile} />
  );
}

function EditProfileForm({
  profile,
  interests: initialInterests,
  onDone
}: {
  profile: ProfileView;
  interests: string[];
  onDone: () => void;
}) {
  const { toast } = useFeedback();
  const save = useSaveProfile();
  const saveInterests = useSaveInterests();
  const catalog = useInterestsCatalog();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender[]>([profile.gender]);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [city, setCity] = useState(profile.city);
  const [height, setHeight] = useState(profile.heightCm ? String(profile.heightCm) : "");
  const [goal, setGoal] = useState<RelationshipGoal[]>([profile.relationshipGoal]);
  const [interests, setInterests] = useState(initialInterests);
  const [errors, setErrors] = useState<{ firstName?: string; height?: string }>({});

  const submit = async () => {
    const heightCm = height.trim() ? Number(height) : null;
    const next = {
      firstName: validateFirstName(firstName) ?? undefined,
      height: heightCm !== null && (!Number.isInteger(heightCm) || heightCm < 120 || heightCm > 230) ? "Entre 120 et 230 cm." : undefined
    };
    setErrors(next);
    if (next.firstName || next.height) {
      return;
    }
    try {
      await save.mutateAsync({
        firstName: firstName.trim(),
        birthdate: profile.birthdate,
        gender: gender[0],
        bio: bio.trim(),
        jobTitle: jobTitle.trim(),
        city: city.trim(),
        heightCm,
        relationshipGoal: goal[0]
      });
      await saveInterests.mutateAsync(interests);
      toast("Profil mis à jour", "success");
      onDone();
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  return (
    <Screen keyboard scroll testID="edit-profile-screen">
      <Header onBack={onDone} title="Mon profil" />
      <TextField error={errors.firstName} label="Prénom" maxLength={30} onChangeText={setFirstName} value={firstName} />
      <Card>
        <Text color="textMuted" variant="caption">
          Âge : {profile.age} ans. La date de naissance ne peut pas être modifiée, pour la sécurité de tous.
        </Text>
      </Card>
      <ChoiceGroup label="Je suis" labels={genderLabels} onChange={setGender} options={genders} value={gender} />
      <TextField hint={`${bio.length}/500`} label="Bio" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
      <View style={{ flexDirection: "row", gap: spacing.sm }}>
        <View style={{ flex: 2 }}>
          <TextField label="Métier" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
        </View>
        <View style={{ flex: 1 }}>
          <TextField
            error={errors.height}
            inputMode="numeric"
            keyboardType="number-pad"
            label="Taille (cm)"
            maxLength={3}
            onChangeText={(v) => setHeight(v.replace(/\D/g, ""))}
            value={height}
          />
        </View>
      </View>
      <TextField hint="Affichée telle quelle sur votre profil." label="Ville" maxLength={60} onChangeText={setCity} value={city} />
      <ChoiceGroup label="Je cherche plutôt" labels={goalLabels} onChange={setGoal} options={goals} value={goal} />
      <Text variant="heading">Centres d’intérêt</Text>
      <InterestPicker catalog={catalog.data} onChange={setInterests} selected={interests} />
      <Button label="Enregistrer" loading={save.isPending || saveInterests.isPending} onPress={submit} testID="edit-save" />
    </Screen>
  );
}
