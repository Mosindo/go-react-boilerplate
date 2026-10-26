import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { OwnProfile, PublicProfile } from "../api";
import { FeedbackProvider } from "../components/Feedback";
import { ProfileForm } from "../components/ProfileForm";
import { SwipeDeck } from "../components/SwipeDeck";
import { WelcomeScreen } from "../screens/WelcomeScreen";
import { ThemeProvider } from "../theme";

const mockSignIn = jest.fn();
const mockSignUp = jest.fn();

jest.mock("@react-navigation/native", () => ({ useNavigation: () => ({ navigate: jest.fn() }) }));
jest.mock("react-native-safe-area-context", () => {
  const { View } = jest.requireActual("react-native");
  return { SafeAreaView: View, SafeAreaProvider: View };
});
jest.mock("../auth/AuthContext", () => ({
  useAuth: () => ({ signIn: mockSignIn, signUp: mockSignUp })
}));
jest.mock("../api", () => ({
  ...jest.requireActual("../api/types"),
  API_BASE_URL: "http://api.test"
}));
jest.mock("../hooks/useQueries", () => ({
  useInterests: () => ({
    isPending: false,
    data: [
      { slug: "travel", label: "Voyages" },
      { slug: "music", label: "Musique" }
    ]
  })
}));

function wrap(ui: React.ReactElement) {
  const client = new QueryClient();
  return render(
    <ThemeProvider>
      <QueryClientProvider client={client}>
        <FeedbackProvider>{ui}</FeedbackProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

beforeEach(() => {
  mockSignIn.mockReset();
  mockSignUp.mockReset();
});

describe("WelcomeScreen", () => {
  it("blocks submission with an invalid email and shows the error", async () => {
    wrap(<WelcomeScreen />);
    fireEvent.changeText(screen.getByTestId("email"), "not-an-email");
    fireEvent.changeText(screen.getByTestId("password"), "Password123");
    fireEvent.press(screen.getByTestId("submit-auth"));
    expect(await screen.findByText("Entrez une adresse email valide.")).toBeTruthy();
    expect(mockSignIn).not.toHaveBeenCalled();
  });

  it("signs in with trimmed credentials", async () => {
    mockSignIn.mockResolvedValue(undefined);
    wrap(<WelcomeScreen />);
    fireEvent.changeText(screen.getByTestId("email"), "me@example.com");
    fireEvent.changeText(screen.getByTestId("password"), "Password123");
    fireEvent.press(screen.getByTestId("submit-auth"));
    await waitFor(() => expect(mockSignIn).toHaveBeenCalledWith("me@example.com", "Password123"));
  });

  it("enforces the password policy when registering and surfaces server errors", async () => {
    mockSignUp.mockRejectedValue(new Error("email already exists"));
    wrap(<WelcomeScreen />);
    fireEvent.press(screen.getByTestId("tab-register"));
    fireEvent.changeText(screen.getByTestId("email"), "me@example.com");
    fireEvent.changeText(screen.getByTestId("password"), "short");
    fireEvent.press(screen.getByTestId("submit-auth"));
    expect(await screen.findByText("8 caractères minimum.")).toBeTruthy();
    expect(mockSignUp).not.toHaveBeenCalled();

    fireEvent.changeText(screen.getByTestId("password"), "Password123");
    fireEvent.press(screen.getByTestId("submit-auth"));
    expect(await screen.findByText("email already exists")).toBeTruthy();
  });
});

describe("ProfileForm", () => {
  it("refuses minors before calling the API", async () => {
    const onSubmit = jest.fn();
    wrap(<ProfileForm submitLabel="Continuer" onSubmit={onSubmit} />);
    fireEvent.changeText(screen.getByTestId("first-name"), "Léa");
    const year = new Date().getFullYear() - 16;
    fireEvent.changeText(screen.getByTestId("birth-date"), `0101${year}`);
    fireEvent.press(screen.getByTestId("gender-woman"));
    fireEvent.press(screen.getByTestId("save-profile"));
    expect(await screen.findByText("Lien est réservé aux personnes majeures.")).toBeTruthy();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits normalised data for an adult", async () => {
    const onSubmit = jest.fn();
    wrap(<ProfileForm submitLabel="Continuer" onSubmit={onSubmit} />);
    fireEvent.changeText(screen.getByTestId("first-name"), "  Léa ");
    fireEvent.changeText(screen.getByTestId("birth-date"), "14071994");
    fireEvent.press(screen.getByTestId("gender-woman"));
    fireEvent.press(screen.getByText("Voyages"));
    fireEvent.press(screen.getByTestId("save-profile"));
    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith({
        firstName: "Léa",
        birthDate: "1994-07-14",
        gender: "woman",
        bio: "",
        jobTitle: "",
        interests: ["travel"]
      })
    );
  });

  it("locks the birth date of an existing profile", () => {
    const existing = {
      firstName: "Léa",
      birthDate: "1994-07-14",
      gender: "woman",
      bio: "",
      jobTitle: "",
      interests: []
    } as unknown as OwnProfile;
    wrap(<ProfileForm initial={existing} submitLabel="Enregistrer" onSubmit={jest.fn()} />);
    expect(screen.getByTestId("birth-date").props.editable).toBe(false);
    expect(screen.getByTestId("birth-date").props.value).toBe("14/07/1994");
  });
});

const profile = (id: string, name: string): PublicProfile => ({
  userId: id,
  firstName: name,
  age: 30,
  gender: "woman",
  bio: "",
  jobTitle: "",
  city: "Paris",
  distanceKm: 5,
  interests: [],
  photos: []
});

describe("SwipeDeck", () => {
  it("likes and passes the top card with the accessible buttons", async () => {
    const onSwipe = jest.fn();
    wrap(
      <SwipeDeck
        profiles={[profile("1", "Alice"), profile("2", "Bea")]}
        onSwipe={onSwipe}
        onInfo={jest.fn()}
      />
    );
    expect(screen.getAllByText("Alice, 30").length).toBeGreaterThan(0);
    fireEvent.press(screen.getByTestId("like-button"));
    await waitFor(() =>
      expect(onSwipe).toHaveBeenCalledWith(expect.objectContaining({ userId: "1" }), "like")
    );
  });

  it("opens the full profile from the info button", () => {
    const onInfo = jest.fn();
    wrap(<SwipeDeck profiles={[profile("1", "Alice")]} onSwipe={jest.fn()} onInfo={onInfo} />);
    fireEvent.press(screen.getByTestId("info-button"));
    expect(onInfo).toHaveBeenCalledWith(expect.objectContaining({ userId: "1" }));
  });
});
