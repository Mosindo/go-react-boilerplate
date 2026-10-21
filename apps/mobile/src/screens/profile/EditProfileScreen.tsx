import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useState } from "react";
import { fetchProfile, saveProfile } from "../../api/profile";
import { ProfileFields } from "../../components/ProfileFields";
import { keys } from "../../hooks/queries";
import { getApproxLocation } from "../../lib/device";
import { draftFromProfile, draftToInput, emptyDraft, validateDraft, type DraftErrors, type ProfileDraft } from "../../lib/profileDraft";
import type { RootStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView, showError, showToast } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Button, Notice } from "../../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;

export function EditProfileScreen({ navigation }: Props) {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.profile, queryFn: fetchProfile });
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [errors, setErrors] = useState<DraftErrors>({});
  const [locating, setLocating] = useState(false);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (query.data && !loaded) {
      setDraft(draftFromProfile(query.data));
      setLoaded(true);
    }
  }, [query.data, loaded]);

  const save = useMutation({
    mutationFn: () => saveProfile(draftToInput(draft)),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      showToast("Profil enregistré.", "success");
      navigation.goBack();
    }
  });

  const useLocation = async () => {
    setLocating(true);
    const loc = await getApproxLocation();
    setLocating(false);
    if (!loc) return showError("Position indisponible. Vérifiez l'autorisation de localisation.");
    setDraft((d) => ({ ...d, latitude: loc.latitude, longitude: loc.longitude, clearLocation: false, city: d.city || loc.city || "" }));
  };

  const submit = () => {
    const found = validateDraft(draft);
    setErrors(found);
    if (Object.keys(found).length === 0) save.mutate();
  };

  if (query.isPending || !loaded)
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  if (query.isError)
    return (
      <ScreenContainer>
        <ErrorView message={query.error.message} onRetry={() => void query.refetch()} />
      </ScreenContainer>
    );

  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="edit-profile">
      {save.isError ? <Notice message={save.error instanceof Error ? save.error.message : "Enregistrement impossible."} /> : null}
      <ProfileFields
        draft={draft}
        errors={errors}
        onChange={(p) => setDraft((d) => ({ ...d, ...p }))}
        onUseLocation={useLocation}
        locating={locating}
      />
      <Button testID="profile-save" label="Enregistrer" loading={save.isPending} onPress={submit} />
    </ScreenContainer>
  );
}
