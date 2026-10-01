import React, { useEffect, useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { profileApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import { GENDERS, type Gender } from "../api/types";
import { InterestPicker } from "../components/InterestPicker";
import { PhotoManager } from "../components/PhotoManager";
import { useAuth } from "../hooks/useAuth";
import { useDeviceLocation } from "../hooks/useDeviceLocation";
import type { RootScreenProps } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Button, Chip, FormField, Input, Notice, Text, spacing } from "../shared/ui";

export default function EditProfileScreen({ navigation }: RootScreenProps<"EditProfile">) {
  const queryClient = useQueryClient();
  const { refreshUser } = useAuth();
  const { data: profile, isLoading, isError, refetch } = useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.getOwn });
  const [firstName, setFirstName] = useState("");
  const [gender, setGender] = useState<Gender>("woman");
  const [bio, setBio] = useState("");
  const [city, setCity] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const location = useDeviceLocation((found) => found && setCity(found));

  useEffect(() => {
    if (profile && !loaded) {
      setLoaded(true);
      setFirstName(profile.firstName);
      setGender(profile.gender);
      setBio(profile.bio);
      setCity(profile.city);
      setInterests(profile.interests.map((i) => i.slug));
    }
  }, [profile, loaded]);

  const save = useMutation({
    mutationFn: () => profileApi.save({ firstName: firstName.trim(), gender, bio, city, interests }),
    onSuccess: async (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.publicProfile(saved.id) });
      await refreshUser();
      showToast("Profil enregistré.");
      navigation.goBack();
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Enregistrement impossible.")
  });

  if (isLoading) {
    return <LoadingView fullScreen />;
  }
  if (isError || !profile) {
    return <ErrorView message="Impossible de charger votre profil." onAction={() => void refetch()} />;
  }

  return (
    <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <FormField label="Photos">
          <PhotoManager onChanged={() => void refreshUser()} />
        </FormField>
        <FormField label="Prénom">
          <Input accessibilityLabel="Prénom" autoCapitalize="words" maxLength={50} onChangeText={setFirstName} value={firstName} />
        </FormField>
        <FormField label="Je suis">
          <View style={styles.row}>
            {GENDERS.map((g) => (
              <Chip key={g.value} label={g.label} onPress={() => setGender(g.value)} selected={gender === g.value} />
            ))}
          </View>
        </FormField>
        <FormField hint={`${bio.length}/500`} label="Bio">
          <Input accessibilityLabel="Bio" maxLength={500} multiline onChangeText={setBio} value={bio} />
        </FormField>
        <FormField label="Centres d'intérêt">
          <InterestPicker onChange={setInterests} selected={interests} />
        </FormField>
        <FormField hint="Seule une distance arrondie est visible par les autres." label="Ville et position">
          <Input accessibilityLabel="Ville" maxLength={80} onChangeText={setCity} value={city} />
          <Button
            label="Actualiser ma position"
            loading={location.state === "working"}
            onPress={() => void location.share()}
            size="sm"
            variant="outline"
          />
          {location.message ? <Notice message={location.message} tone={location.state === "done" ? "success" : "warning"} /> : null}
        </FormField>
        <Text tone="subtle" variant="caption">
          Âge : {profile.age} ans (la date de naissance ne peut pas être modifiée).
        </Text>
        {error ? <Notice message={error} tone="danger" /> : null}
        <Button
          label="Enregistrer"
          loading={save.isPending}
          onPress={() => (firstName.trim() ? save.mutate() : setError("Le prénom est obligatoire."))}
        />
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { padding: spacing.lg, gap: spacing.lg, paddingBottom: spacing.xxxl },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
