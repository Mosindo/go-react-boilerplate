import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { saveInterests, saveLocation, saveProfile } from "../api/dating";
import { InterestPicker } from "../components/forms/InterestPicker";
import { LocationStep } from "../components/forms/LocationStep";
import { ProfileBasicsForm, type BasicsValues } from "../components/forms/ProfileBasicsForm";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Section } from "../shared/layout";
import { spacing } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;

export default function EditProfileScreen({ navigation }: Props) {
  const { self, refreshSelf } = useAuth();
  const [interestIds, setInterestIds] = useState<number[]>(self?.interests.map((interest) => interest.id) ?? []);

  const save = useMutation({
    mutationFn: async (values: BasicsValues) => {
      await saveProfile(values);
      await saveInterests(interestIds);
    },
    onSuccess: async () => {
      await refreshSelf();
      showToast("Profil mis à jour", "success");
      navigation.goBack();
    }
  });

  const relocate = useMutation({
    mutationFn: saveLocation,
    onSuccess: async () => {
      await refreshSelf();
      showToast("Position mise à jour", "success");
    }
  });

  return (
    <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled" testID="edit-profile-screen">
        <Section title="Informations">
          <ProfileBasicsForm
            busy={save.isPending}
            error={save.isError ? describeError(save.error) : null}
            initial={self?.profile}
            onSubmit={(values) => save.mutate(values)}
            submitLabel="Enregistrer"
            withBio
          />
        </Section>
        <Section title="Centres d'intérêt">
          <InterestPicker onChange={setInterestIds} selectedIds={interestIds} />
        </Section>
        <Section title="Position">
          <View>
            <LocationStep
              busy={relocate.isPending}
              currentCity={self?.profile?.city}
              error={relocate.isError ? describeError(relocate.error) : null}
              onLocated={(location) => relocate.mutate(location)}
              submitLabel="Mettre à jour ma position"
            />
          </View>
        </Section>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { padding: spacing.xl, paddingBottom: spacing.xxxl }
});
