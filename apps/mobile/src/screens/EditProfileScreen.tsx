import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { saveLocation, saveProfile } from "../api/platform";
import { errorMessage } from "../api/client";
import type { Gender, OwnProfile } from "../api/types";
import { InterestPicker } from "../features/profile/InterestPicker";
import { useOwnProfile } from "../hooks/queries";
import { GENDERS, GENDER_LABELS, formatBirthDate } from "../lib/format";
import { getApproximateLocation } from "../lib/location";
import { queryKeys } from "../lib/queryClient";
import { showToast } from "../shared/feedback";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, spacing } from "../shared/ui";
import { validateFirstName } from "../lib/validation";

export default function EditProfileScreen() {
  const query = useOwnProfile();
  if (query.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (!query.data) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger votre profil." onAction={() => void query.refetch()} />
      </ScreenContainer>
    );
  }
  return <EditForm profile={query.data} />;
}

function EditForm({ profile }: { profile: OwnProfile }) {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [city, setCity] = useState(profile.city ?? "");
  const [interestIds, setInterestIds] = useState(profile.interestIds);
  const [nameError, setNameError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () =>
      saveProfile({
        firstName: firstName.trim(),
        birthDate: profile.birthDate,
        gender,
        bio: bio.trim(),
        city: city.trim() || null,
        interestIds
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      showToast("Profil mis à jour.");
      navigation.goBack();
    },
    onError: (error) => setFormError(errorMessage(error))
  });

  const relocate = useMutation({
    mutationFn: async () => {
      const result = await getApproximateLocation();
      if (result.status !== "ok") {
        return result.status;
      }
      await saveLocation({ latitude: result.latitude, longitude: result.longitude });
      if (result.city) {
        setCity(result.city);
      }
      return "ok" as const;
    },
    onSuccess: async (status) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      showToast(status === "ok" ? "Position mise à jour." : "Position indisponible. Vérifiez l'autorisation de localisation.");
    },
    onError: (error) => setFormError(errorMessage(error))
  });

  function submit() {
    const error = validateFirstName(firstName);
    setNameError(error);
    setFormError(null);
    if (!error) {
      save.mutate();
    }
  }

  return (
    <ScreenContainer>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          {formError ? <Notice message={formError} tone="danger" /> : null}
          <Input error={nameError} label="Prénom" onChangeText={setFirstName} testID="edit-name" value={firstName} />
          <Input
            editable={false}
            hint="La date de naissance ne peut pas être modifiée."
            label="Date de naissance"
            value={formatBirthDate(profile.birthDate)}
          />
          <FormField label="Je suis">
            <View style={styles.wrap}>
              {GENDERS.map((g) => (
                <Chip key={g} label={GENDER_LABELS[g]} onPress={() => setGender(g)} selected={gender === g} />
              ))}
            </View>
          </FormField>
          <Input hint={`${bio.length}/500`} label="Bio" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
          <Input label="Ville" maxLength={80} onChangeText={setCity} value={city} />
          <Button
            label={profile.latitude === null ? "Activer ma position" : "Actualiser ma position"}
            loading={relocate.isPending}
            onPress={() => relocate.mutate()}
            variant="outline"
          />
          <FormField hint="Jusqu'à 10." label="Centres d'intérêt">
            <InterestPicker onChange={setInterestIds} value={interestIds} />
          </FormField>
          <Button fullWidth label="Enregistrer" loading={save.isPending} onPress={submit} size="lg" testID="edit-save" />
        </ScrollView>
      </KeyboardAvoidingView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { gap: spacing.lg, paddingBottom: spacing.xxxl },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
