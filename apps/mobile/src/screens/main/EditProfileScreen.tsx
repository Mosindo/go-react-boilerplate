import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AppText, Banner, Button, ErrorState, Loading, Screen } from "../../components/ui";
import PhotosEditor from "../../components/PhotosEditor";
import {
  AboutSection,
  IdentitySection,
  PreferencesSection,
  PrivacySection,
  draftFrom,
  toInput,
  validateDraft,
  type Draft,
} from "../../components/ProfileSections";
import { profileApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { useMyProfile } from "../../hooks";
import type { MainStackParams } from "../../navigation/types";

export default function EditProfileScreen({
  navigation,
}: NativeStackScreenProps<MainStackParams, "EditProfile">) {
  const qc = useQueryClient();
  const q = useMyProfile();
  const [draft, setDraft] = useState<Draft | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: (d: Draft) => profileApi.save(toInput(d)),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["profile"] });
      // Preferences changed: the discovery deck must be recomputed.
      void qc.invalidateQueries({ queryKey: ["likes"] });
      navigation.goBack();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  if (q.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (q.isError || !q.data)
    return (
      <Screen>
        <ErrorState message="Impossible de charger votre profil." onRetry={() => void q.refetch()} />
      </Screen>
    );

  const current = draft ?? draftFrom(q.data);
  const set = (patch: Partial<Draft>) => {
    setDraft({ ...current, ...patch });
    setErrors({});
  };

  const submit = () => {
    const found = validateDraft(current, ["identity", "about", "preferences"]);
    if (Object.keys(found).length > 0) {
      setErrors(found);
      setError("Certains champs sont à corriger.");
      return;
    }
    setError(null);
    save.mutate(current);
  };

  return (
    <Screen scroll>
      {error ? <Banner message={error} /> : null}
      <AppText variant="heading">Photos</AppText>
      <PhotosEditor photos={q.data.photos} />
      <AppText variant="heading">Mes informations</AppText>
      <IdentitySection draft={current} set={set} errors={errors} hadLocation={q.data.hasLocation} />
      <AboutSection draft={current} set={set} errors={errors} />
      <AppText variant="heading">Préférences de rencontre</AppText>
      <PreferencesSection draft={current} set={set} errors={errors} />
      <AppText variant="heading">Confidentialité</AppText>
      <PrivacySection draft={current} set={set} errors={errors} />
      <Button testID="save-profile" title="Enregistrer" onPress={submit} loading={save.isPending} />
    </Screen>
  );
}
