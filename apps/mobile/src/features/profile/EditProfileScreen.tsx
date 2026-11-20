import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { profileApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { splitIsoDate } from "../../lib/validation";
import type { AppStackParamList } from "../../navigation/types";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { Button, Header, LoadingState, Screen, SectionTitle, showToast, Text } from "../../ui";
import { AboutFields, BasicsFields, InterestPicker, type AboutValue, type BasicsValue } from "./ProfileFields";
import { useInterests, useMyProfile } from "./queries";

type Props = NativeStackScreenProps<AppStackParamList, "EditProfile">;

export function EditProfileScreen({ navigation }: Props) {
  const { spacing } = useTheme();
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const interests = useInterests();
  const [basics, setBasics] = useState<BasicsValue | null>(null);
  const [about, setAbout] = useState<AboutValue | null>(null);
  const [interestIds, setInterestIds] = useState<number[]>([]);

  useEffect(() => {
    const p = profile.data;
    if (p && !basics) {
      setBasics({ firstName: p.firstName, ...splitIsoDate(p.birthdate), gender: p.gender });
      setAbout({ bio: p.bio, jobTitle: p.jobTitle, city: p.city, relationshipGoal: p.relationshipGoal });
      setInterestIds(p.interests.map((i) => i.id));
    }
  }, [profile.data, basics]);

  const save = useMutation({
    mutationFn: async () => {
      const p = profile.data;
      if (!p || !basics || !about || !basics.gender) {
        throw new Error("Profil indisponible.");
      }
      return profileApi.save({
        firstName: basics.firstName.trim(),
        birthdate: p.birthdate,
        gender: basics.gender,
        bio: about.bio.trim(),
        jobTitle: about.jobTitle.trim(),
        city: about.city.trim(),
        heightCm: p.heightCm,
        relationshipGoal: about.relationshipGoal,
        interestIds
      });
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(keys.profile, saved);
      showToast("Profil mis à jour.", "success");
      navigation.goBack();
    }
  });

  if (!basics || !about) {
    return (
      <Screen>
        <Header back title="Modifier mon profil" />
        <LoadingState />
      </Screen>
    );
  }
  const nameError = basics.firstName.trim() ? null : "Indique ton prénom.";
  return (
    <Screen keyboard scroll header={<Header back title="Modifier mon profil" />}>
      <BasicsFields birthdateLocked errors={{ firstName: nameError }} onChange={setBasics} value={basics} />
      <AboutFields onChange={setAbout} value={about} />
      <View style={{ gap: spacing.sm }}>
        <SectionTitle>Centres d’intérêt</SectionTitle>
        {interests.data ? <InterestPicker interests={interests.data} onChange={setInterestIds} selected={interestIds} /> : <LoadingState />}
      </View>
      {save.isError ? <Text tone="danger">{errorMessage(save.error)}</Text> : null}
      <Button disabled={!!nameError} label="Enregistrer" loading={save.isPending} onPress={() => save.mutate()} testID="edit-save" />
    </Screen>
  );
}
