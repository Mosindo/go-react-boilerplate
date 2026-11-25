import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/profile";
import { ProfileFields } from "../../components/ProfileFields";
import { formToInput, profileToForm, type ProfileFormErrors, type ProfileFormState } from "../../components/profileForm";
import { useMyProfile, useSetMyProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { Button, Header, LoadingState, Screen, Text, showToast } from "../../ui";

export function EditProfileScreen({ navigation }: AppScreenProps<"EditProfile">) {
  const me = useMyProfile();
  const setMyProfile = useSetMyProfile();
  const [form, setForm] = useState<ProfileFormState | null>(null);
  const [errors, setErrors] = useState<ProfileFormErrors>({});
  const current = form ?? profileToForm(me.data?.profile);

  const save = useMutation({
    mutationFn: async () => {
      const { input, errors: next } = formToInput(current);
      setErrors(next);
      if (!input) {
        return null;
      }
      return profileApi.update(input);
    },
    onSuccess: (profile) => {
      if (profile) {
        setMyProfile(profile);
        showToast("Profil mis à jour.", "success");
        navigation.goBack();
      }
    }
  });

  if (!me.data) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }

  return (
    <Screen
      scroll
      testID="edit-profile-screen"
      footer={
        <>
          {save.error ? (
            <Text variant="small" color="danger" align="center">
              {errorMessage(save.error)}
            </Text>
          ) : null}
          <Button label="Enregistrer" onPress={() => save.mutate()} loading={save.isPending} testID="edit-profile-save" />
        </>
      }
    >
      <Header title="Modifier mon profil" onBack={navigation.goBack} />
      <ProfileFields form={current} errors={errors} onChange={(p) => setForm({ ...current, ...p })} sections={["basics", "about"]} />
    </Screen>
  );
}
