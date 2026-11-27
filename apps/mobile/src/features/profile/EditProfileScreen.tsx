import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, MyProfile } from "../../api/types";
import { useMyProfile } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { spacing } from "../../theme/palette";
import { Button, Header, InlineError, Loading, Screen, Section, toast } from "../../ui";
import type { AppScreenProps } from "../../navigation/types";
import { AboutForm } from "./components/AboutForm";
import { BasicsForm } from "./components/BasicsForm";
import { PhotoManager } from "./components/PhotoManager";
import { basicsFromProfile, hasErrors, validateBasics, type BasicsErrors } from "./profileForm";

function EditForm({ profile, onSaved }: { profile: MyProfile; onSaved: () => void }) {
  const [basics, setBasics] = useState(() => basicsFromProfile(profile));
  const [errors, setErrors] = useState<BasicsErrors>({});
  const [bio, setBio] = useState(profile.bio);
  const [interestIds, setInterestIds] = useState(profile.interests.map((i) => i.id));

  const save = useMutation({
    mutationFn: async () => {
      const { errors: next, birthdate } = validateBasics(basics);
      setErrors(next);
      if (hasErrors(next) || !birthdate) {
        return null;
      }
      await profileApi.save({ firstName: basics.firstName.trim(), birthdate, gender: basics.gender as Gender, bio, city: basics.city.trim() });
      return profileApi.saveInterests(interestIds);
    },
    onSuccess: (updated) => {
      if (updated) {
        queryClient.setQueryData(keys.myProfile, updated);
        toast("Profil enregistré.", "success");
        onSaved();
      }
    }
  });

  return (
    <View style={styles.content}>
      <Section title="Photos">
        <PhotoManager photos={profile.photos} />
      </Section>
      <Section title="Informations">
        <BasicsForm value={basics} onChange={setBasics} errors={errors} />
      </Section>
      <Section title="À propos">
        <AboutForm bio={bio} interestIds={interestIds} onBioChange={setBio} onInterestsChange={setInterestIds} />
      </Section>
      <InlineError message={save.error ? errorMessage(save.error) : null} />
      <Button label="Enregistrer" size="lg" loading={save.isPending} onPress={() => save.mutate()} style={styles.save} testID="edit-save" />
    </View>
  );
}

export function EditProfileScreen({ navigation }: AppScreenProps<"EditProfile">) {
  const profile = useMyProfile();
  return (
    <Screen scroll keyboard>
      <Header title="Modifier mon profil" onBack={() => navigation.goBack()} />
      {profile.data ? <EditForm profile={profile.data} onSaved={() => navigation.goBack()} /> : <Loading />}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 560, width: "100%", alignSelf: "center" },
  save: { marginTop: spacing.xl }
});
