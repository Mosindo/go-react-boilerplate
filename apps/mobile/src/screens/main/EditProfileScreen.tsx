import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React from "react";
import { ProfileForm } from "../../components/ProfileForm";
import { Screen } from "../../components/Screen";
import { ErrorView, LoadingView } from "../../components/StateViews";
import { useProfile } from "../../hooks/queries";
import type { RootStackParamList } from "../../navigation/types";
import { useToast } from "../../state/ToastProvider";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;

export default function EditProfileScreen({ navigation }: Props) {
  const toast = useToast();
  const query = useProfile();
  if (query.isLoading) return <LoadingView />;
  if (query.isError || !query.data) return <ErrorView onRetry={() => void query.refetch()} />;
  return (
    <Screen edges={["bottom"]} scroll>
      <ProfileForm
        initial={query.data}
        onSaved={() => {
          toast.show("Profil mis à jour.", "success");
          navigation.goBack();
        }}
        submitLabel="Enregistrer"
      />
    </Screen>
  );
}
