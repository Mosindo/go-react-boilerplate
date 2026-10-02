import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import React from "react";

import { AuthProvider } from "../auth/AuthContext";
import { RegisterScreen } from "../screens/auth/AuthScreens";
import { FeedbackProvider } from "../ui/Feedback";
import { Chip } from "../ui/Chip";

jest.mock("@expo/vector-icons", () => ({ Ionicons: () => null }));
jest.mock("react-native-safe-area-context", () => {
  const { View } = jest.requireActual("react-native");
  return { SafeAreaView: View, SafeAreaProvider: View };
});

function renderWithProviders(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: 0 } } });
  return render(
    <QueryClientProvider client={client}>
      <FeedbackProvider>
        <AuthProvider>{ui}</AuthProvider>
      </FeedbackProvider>
    </QueryClientProvider>,
  );
}

afterEach(() => jest.useRealTimers());

describe("RegisterScreen", () => {
  const navigation = { navigate: jest.fn() } as never;
  const route = { key: "r", name: "Register" } as never;

  it("blocks submission and explains invalid input without calling the API", async () => {
    const fetchSpy = jest.fn();
    global.fetch = fetchSpy as unknown as typeof fetch;
    renderWithProviders(<RegisterScreen navigation={navigation} route={route} />);

    fireEvent.changeText(screen.getByTestId("register-email"), "not-an-email");
    fireEvent.changeText(screen.getByTestId("register-password"), "short");
    fireEvent.press(screen.getByTestId("register-submit"));

    await waitFor(() => expect(screen.getByText("Adresse e-mail invalide.")).toBeTruthy());
    expect(screen.getByText("8 caractères minimum.", { exact: false })).toBeTruthy();
    expect(fetchSpy).not.toHaveBeenCalled();
  });
});

describe("Chip", () => {
  it("exposes its selected state to assistive technology", () => {
    const onPress = jest.fn();
    render(<Chip label="Randonnée" selected onPress={onPress} testID="chip" />);
    const chip = screen.getByTestId("chip");
    expect(chip.props.accessibilityState).toMatchObject({ selected: true });
    fireEvent.press(chip);
    expect(onPress).toHaveBeenCalledTimes(1);
  });
});
