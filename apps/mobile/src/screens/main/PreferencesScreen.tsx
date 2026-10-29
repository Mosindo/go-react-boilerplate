import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React from "react";
import { PreferencesForm } from "../../components/PreferencesForm";
import { Screen } from "../../components/Screen";
import { ErrorView, LoadingView } from "../../components/StateViews";
import { usePreferences } from "../../hooks/queries";
import type { RootStackParamList } from "../../navigation/types";
import { useToast } from "../../state/ToastProvider";

type Props = NativeStackScreenProps<RootStackParamList, "Preferences">;

export default function PreferencesScreen({ navigation }: Props) {
  const toast = useToast();
  const query = usePreferences();
  if (query.isLoading) return <LoadingView />;
  if (query.isError || !query.data) return <ErrorView onRetry={() => void query.refetch()} />;
  return (
    <Screen edges={["bottom"]} scroll>
      <PreferencesForm
        initial={query.data}
        onSaved={() => {
          toast.show("Préférences enregistrées.", "success");
          navigation.goBack();
        }}
        submitLabel="Enregistrer"
      />
    </Screen>
  );
}
