package desktop

import (
	"strings"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/egoist/mygo/ui"
)

// Steps of the Cloudflare sign-in.
const (
	stepSignIn = iota
	stepBucket
)

// openConnect opens the add-or-edit connection page.
func (a *App) openConnect(id string) {
	state := &a.connect
	state.open = true
	state.editing = id
	state.err = nil
	state.notice = ""
	state.testing = false
	state.busy = false
	state.signIn = service.CloudflareSignIn{}
	state.accounts = nil
	state.buckets = nil
	state.step = stepSignIn

	if id == "" {
		state.profile = core.Profile{ProviderType: "r2", CredentialSource: "keychain"}
		state.name = ""
		state.accountID = ""
		state.bucket = ""
		state.jurisdiction = "default"
		state.publicURL = ""
		state.readOnly = false
		state.credentialSource = "keychain"
		state.accessKeyID = ""
		state.secretKey = ""
		state.keychainSaved = false
		return
	}
	profile := a.profileOf(id)
	if profile == nil {
		state.open = false
		return
	}
	state.profile = *profile
	state.name = profile.Name
	state.accountID = profile.AccountID
	state.bucket = profile.Bucket
	state.jurisdiction = profile.Jurisdiction
	state.publicURL = profile.PublicURL
	state.readOnly = profile.ReadOnly
	state.credentialSource = profile.CredentialSource
	state.accessKeyID = ""
	state.secretKey = ""
	state.keychainSaved = profile.CredentialSource != "environment"
	if profile.CredentialSource == "cloudflare" {
		a.loadCloudflareStatus(profile.ID)
	}
}

// closeConnect leaves the connection page.
func (a *App) closeConnect() {
	if a.connect.signIn.SessionID != "" {
		session := a.connect.signIn.SessionID
		go func() { _ = a.svc.CancelCloudflareSignIn(a.ctx, session) }()
	}
	a.connect = connectState{}
}

// connectOverlay draws the connection dialog: the providers on the left,
// the chosen one's form on the right.
func (a *App) connectOverlay(c *ui.Context) {
	state := &a.connect
	if !state.open {
		return
	}
	title := a.t("connection.titleNew")
	if state.editing != "" {
		title = a.t("connection.titleEdit")
	}
	a.dialog(c, 820, false, true, a.closeConnect, func() {
		ui.Column(c).Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(a.pal.border).Children(func() {
			a.dialogHeader(c, iconDatabase, a.pal.primarySoft, a.pal.primaryText, title,
				a.t("connection.flow.desc"), nil, a.closeConnect)
		})
		ui.Row(c).Height(520).Shrink(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.providerList(c)
			ui.Scroll(c).Grow(1).MinWidth(0).Children(func() {
				ui.Column(c).Gap(14).Padding(18, 22, 22, 22).Children(func() {
					a.connectForm(c)
				})
			})
		})
		a.connectFooter(c)
	})
}

// providerList lists the storage services a connection can use.
func (a *App) providerList(c *ui.Context) {
	ui.Column(c).Width(220).Shrink(0).Gap(1).Padding(10, 8).Background(a.pal.sidebar).
		BorderWidth(0, 1, 0, 0).BorderColor(a.pal.border).Children(func() {
		ui.Text(c, a.t("connection.provider")).FontSize(11).FontWeight(600).TextColor(a.pal.faint).
			Padding(4, 8, 6, 8)
		ui.Row(c).Height(34).PaddingX(8).Gap(9).Radius(8).AlignItems(ui.Center).
			Background(a.pal.primary).Children(func() {
			a.providerTile(c, "r2", 20)
			ui.Text(c, a.t("connection.r2")).FontSize(12.5).FontWeight(600).TextColor(a.pal.onPrimary).
				Grow(1).SingleLine()
			ui.Icon(c, iconCheck).FontSize(12).TextColor(a.pal.onPrimary)
		})
	})
}

// connectForm shows the manual fields or the Cloudflare sign-in.
func (a *App) connectForm(c *ui.Context) {
	state := &a.connect
	ui.Row(c).Gap(8).AlignItems(ui.End).Children(func() {
		ui.Text(c, a.t("connection.r2")).FontSize(14).FontWeight(650)
		ui.Text(c, a.t("connection.r2Note")).FontSize(11.5).TextColor(a.pal.faint).SingleLine()
	})
	if state.profile.CredentialSource == "cloudflare" {
		a.cloudflareForm(c)
	} else {
		a.manualForm(c)
	}
	if state.err != nil {
		a.notice(c, a.pal.destructiveBg, a.pal.destructive, a.errorText(state.err))
	}
	if state.notice != "" {
		a.notice(c, a.pal.successBg, a.pal.successText, state.notice)
	}
}

// connectFooter carries the connection's actions.
func (a *App) connectFooter(c *ui.Context) {
	state := &a.connect
	a.dialogFooter(c, func() {
		if state.editing != "" {
			label := a.t("connection.delete")
			remove := a.plainButton(c, label, 0, 30).PaddingX(10)
			if remove.Hovered() {
				remove.Background(a.pal.destructiveBg)
			}
			remove.Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconTrash).FontSize(14).TextColor(a.pal.destructive)
					ui.Text(c, label).FontSize(13).TextColor(a.pal.destructive)
				})
			})
			if remove.Clicked() {
				a.askRemoveProfile()
			}
		}
		if a.secondaryButton(c, nil, a.t("connection.test")).Height(30).Disabled(state.testing).Clicked() {
			a.testConnect()
		}
		if state.testing {
			ui.Spinner(c).Size(12, 12).Label(a.t("connection.testing"))
		}
		ui.Spacer(c)
		if a.secondaryButton(c, nil, a.t("common.cancel")).Height(30).Clicked() {
			a.closeConnect()
			return
		}
		if a.primaryAction(c, nil, a.t("common.save"), false).Height(30).Clicked() {
			a.saveConnect()
		}
	})
}

// formField is a labeled control with an optional hint under it.
func (a *App) formField(c *ui.Context, label, hint string, hintColor ui.Color, control func()) {
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(a.pal.muted)
		control()
		if hint != "" {
			ui.Text(c, hint).FontSize(11.5).TextColor(hintColor)
		}
	})
}

// formPair puts two fields side by side.
func (a *App) formPair(c *ui.Context, left, right func()) {
	ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Children(left)
		ui.Column(c).Grow(1).MinWidth(0).Children(right)
	})
}

// readOnlyToggle switches a connection to read-only, with what that means.
func (a *App) readOnlyToggle(c *ui.Context, value *bool, title, hint string) {
	card := ui.Row(c).Gap(10).Padding(12).Radius(10).AlignItems(ui.Start).Border(1, a.pal.border)
	if *value {
		card.Background(a.pal.warningBg)
	}
	card.Children(func() {
		ui.Switch(c, value).Label(title)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(12.5).FontWeight(500)
			ui.Text(c, hint).FontSize(11.5).TextColor(a.pal.faint)
		})
	})
}

// manualForm is the S3 key form.
func (a *App) manualForm(c *ui.Context) {
	state := &a.connect
	a.formField(c, a.t("connection.name"), "", a.pal.faint, func() {
		ui.TextInput(c, &state.name).Placeholder(a.t("connection.r2")).FillWidth()
	})
	sourceHint := a.t("connection.keychainHint")
	if state.credentialSource == "environment" {
		sourceHint = a.t("connection.environmentHint")
	}
	a.formField(c, a.t("connection.credentialSource"), sourceHint, a.pal.faint, func() {
		sources := []string{"keychain", "environment"}
		labels := []string{a.t("connection.keychain"), a.t("connection.environment")}
		index := 0
		for i, name := range sources {
			if name == state.credentialSource {
				index = i
			}
		}
		ui.Row(c).Children(func() {
			if a.segments(c, &index, a.t("connection.credentialSource"), labels) &&
				index >= 0 && index < len(sources) {
				state.credentialSource = sources[index]
			}
		})
	})
	a.formPair(c, func() {
		invalid := len(state.accountID) > 0 && len(state.accountID) != 32
		hint, color := "", a.pal.faint
		if invalid {
			hint, color = a.t("connection.accountIdError"), a.pal.destructive
		}
		a.formField(c, a.t("connection.accountId"), hint, color, func() {
			input := ui.TextInput(c, &state.accountID).Placeholder(a.t("connection.accountId")).FillWidth()
			if invalid {
				input.Border(1, a.pal.destructive)
			}
		})
	}, func() {
		a.formField(c, a.t("connection.jurisdiction"), "", a.pal.faint, func() {
			options := []string{"default", "eu", "fedramp"}
			current := state.jurisdiction
			if current == "" {
				current = "default"
			}
			if ui.Select(c, &current, options).Label(a.t("connection.jurisdiction")).FillWidth().Changed() {
				state.jurisdiction = current
			}
		})
	})
	if state.credentialSource == "keychain" {
		placeholder := func(label string) string {
			if state.keychainSaved && state.editing != "" {
				return a.t("connection.savedInKeychain")
			}
			return label
		}
		a.formPair(c, func() {
			a.formField(c, a.t("connection.accessKeyId"), "", a.pal.faint, func() {
				ui.TextInput(c, &state.accessKeyID).Placeholder(placeholder(a.t("connection.accessKeyId"))).
					FillWidth()
			})
		}, func() {
			a.formField(c, a.t("connection.secretAccessKey"), "", a.pal.faint, func() {
				ui.TextInput(c, &state.secretKey).Placeholder(placeholder(a.t("connection.secretAccessKey"))).
					Password().FillWidth()
			})
		})
	}
	a.settingsDivider(c)
	a.formField(c, a.t("connection.bucket"), a.t("connection.bucketHint"), a.pal.faint, func() {
		ui.TextInput(c, &state.bucket).Placeholder("—").FillWidth()
	})
	a.publicURLField(c, &state.publicURL)
	a.readOnlyToggle(c, &state.readOnly, a.t("connection.readOnly"), a.t("connection.readOnlyHint"))
	if state.editing != "" {
		ui.Row(c).Children(func() {
			if a.secondaryButton(c, nil, a.t("connection.cloudflare.continue")).Clicked() {
				state.profile.CredentialSource = "cloudflare"
				state.profile.Bucket = state.bucket
				a.beginCloudflareSignIn()
			}
		})
	}
}

// publicURLField takes the address objects are served from publicly.
func (a *App) publicURLField(c *ui.Context, value *string) {
	invalid := *value != "" && !strings.HasPrefix(*value, "https://")
	hint, color := a.t("connection.publicUrlHint"), a.pal.faint
	if invalid {
		hint, color = a.t("connection.publicUrlError"), a.pal.destructive
	}
	a.formField(c, a.t("connection.publicUrl"), hint, color, func() {
		input := ui.TextInput(c, value).Placeholder("https://cdn.example.com").FillWidth()
		if invalid {
			input.Border(1, a.pal.destructive)
		}
	})
}

// cloudflareForm is the sign-in flow of a Cloudflare connection.
func (a *App) cloudflareForm(c *ui.Context) {
	state := &a.connect
	switch {
	case state.signInBusy:
		ui.Column(c).Gap(10).Padding(14, 16).Radius(10).Border(1, a.pal.border).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(12, 12)
				ui.Text(c, a.t("connection.cloudflare.waiting")).FontSize(12.5).FontWeight(600)
			})
			ui.Text(c, a.t("connection.cloudflare.waitingHint")).FontSize(12).TextColor(a.pal.muted)
			if state.signIn.AuthURL != "" {
				a.formField(c, a.t("connection.cloudflare.authUrl"), "", a.pal.faint, func() {
					ui.Text(c, state.signIn.AuthURL).FontSize(11.5).Font("monospace").
						TextColor(a.pal.muted).Selectable().SingleLine()
				})
				ui.Row(c).Gap(8).Children(func() {
					if a.secondaryButton(c, iconExternalLink, a.t("connection.cloudflare.openBrowser")).Clicked() {
						a.openExternally(state.signIn.AuthURL)
					}
					if a.secondaryButton(c, nil, a.t("connection.cloudflare.cancel")).Clicked() {
						a.cancelSignIn()
					}
				})
			}
		})
	case len(state.accounts) > 0:
		a.formPair(c, func() {
			a.formField(c, a.t("connection.cloudflare.account"), "", a.pal.faint, func() {
				options := make([]string, 0, len(state.accounts))
				for _, account := range state.accounts {
					options = append(options, account.Name)
				}
				current := options[0]
				for i, account := range state.accounts {
					if account.ID == state.accountID2 {
						current = options[i]
					}
				}
				if ui.Select(c, &current, options).Label(a.t("connection.cloudflare.account")).FillWidth().Changed() {
					for _, account := range state.accounts {
						if account.Name == current && account.ID != state.accountID2 {
							state.accountID2 = account.ID
							state.buckets = nil
							a.loadCloudflareBuckets()
							return
						}
					}
				}
			})
		}, func() {
			a.formField(c, a.t("connection.cloudflare.bucket"), a.t("connection.cloudflare.bucketHint"), a.pal.faint, func() {
				if len(state.buckets) == 0 {
					ui.Text(c, a.t("connection.cloudflare.noBuckets")).FontSize(12).TextColor(a.pal.muted).Height(30)
					return
				}
				options := make([]string, 0, len(state.buckets))
				for _, bucket := range state.buckets {
					options = append(options, bucket.Name)
				}
				current := state.bucket2
				if current == "" {
					current = options[0]
				}
				if ui.Select(c, &current, options).Label(a.t("connection.cloudflare.bucket")).FillWidth().Changed() {
					state.bucket2 = current
				}
			})
		})
		a.formField(c, a.t("connection.name"), "", a.pal.faint, func() {
			ui.TextInput(c, &state.name).Placeholder(a.t("connection.r2")).FillWidth()
		})
		a.publicURLField(c, &state.publicURL)
		a.readOnlyToggle(c, &state.readOnly2, a.t("connection.cloudflare.readOnly"),
			a.t("connection.cloudflare.readOnlyHint"))
		ui.Text(c, a.t("connection.cloudflare.tokenHint")).FontSize(11.5).TextColor(a.pal.faint)
		ui.Row(c).Gap(8).Children(func() {
			if a.primaryAction(c, nil, a.t("connection.cloudflare.connect"), false).Clicked() {
				a.connectCloudflare()
			}
			if a.secondaryButton(c, nil, a.t("connection.cloudflare.useKeys")).Clicked() {
				state.profile.CredentialSource = "keychain"
				state.accounts = nil
				state.buckets = nil
			}
		})
	case state.editing != "" && state.status.Authorized:
		ui.Column(c).Gap(8).Padding(14, 16).Radius(10).Border(1, a.pal.border).Children(func() {
			ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconCheck).FontSize(13).TextColor(a.pal.successText)
				ui.Text(c, a.t("connection.cloudflare.authorized")).FontSize(12.5).FontWeight(600)
			})
			ui.Text(c, a.tf("connection.cloudflare.authorizedNote", "token", state.status.TokenName)).
				FontSize(12).TextColor(a.pal.muted)
			ui.Row(c).Gap(8).Children(func() {
				if a.secondaryButton(c, iconRefresh, a.t("connection.cloudflare.reauthorize")).Clicked() {
					a.reauthorizeCloudflare()
				}
				if a.secondaryButton(c, nil, a.t("connection.cloudflare.disconnect")).Clicked() {
					a.askDisconnectCloudflare()
				}
			})
		})
	default:
		ui.Column(c).Gap(10).Padding(14, 16).Radius(10).Border(1, a.pal.border).Children(func() {
			ui.Text(c, a.t("connection.cloudflare.signInDesc")).FontSize(12.5).LineHeight(1.5).
				TextColor(a.pal.muted)
			ui.Text(c, a.t("connection.cloudflare.prereq")).FontSize(11.5).TextColor(a.pal.faint)
			ui.Row(c).Children(func() {
				if a.primaryAction(c, nil, a.t("connection.cloudflare.continue"), false).Clicked() {
					a.beginCloudflareSignIn()
				}
			})
		})
	}
	if state.signInErr != nil {
		a.notice(c, a.pal.destructiveBg, a.pal.destructive, a.errorText(state.signInErr))
	}
}

// saveConnect stores the connection.
func (a *App) saveConnect() {
	state := &a.connect
	profile := state.profile
	profile.Name = strings.TrimSpace(state.name)
	profile.AccountID = strings.TrimSpace(state.accountID)
	profile.Bucket = strings.TrimSpace(state.bucket)
	profile.Jurisdiction = state.jurisdiction
	profile.PublicURL = strings.TrimSpace(state.publicURL)
	profile.ReadOnly = state.readOnly
	if profile.CredentialSource != "cloudflare" {
		profile.CredentialSource = state.credentialSource
	}
	if profile.Name == "" {
		state.err = core.Invalid("Enter a connection name")
		return
	}
	state.busy = true
	state.err = nil
	input := service.ProfileInput{
		Profile: profile, AccessKeyID: state.accessKeyID, SecretAccessKey: state.secretKey,
	}
	go func() {
		saved, err := a.svc.SaveProfile(a.ctx, input)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.err = err
				return
			}
			a.closeConnect()
			a.loadProfiles()
			if a.prefs.Quick.ProfileID == "" {
				a.prefs.Quick.ProfileID = saved.ID
				a.savePrefs()
			}
			a.toastCurrent(a.t("connection.saved"))
		})
	}()
}

// testConnect checks the settings without saving them.
func (a *App) testConnect() {
	state := &a.connect
	profile := state.profile
	profile.Name = strings.TrimSpace(state.name)
	profile.AccountID = strings.TrimSpace(state.accountID)
	profile.Bucket = strings.TrimSpace(state.bucket)
	profile.Jurisdiction = state.jurisdiction
	profile.PublicURL = strings.TrimSpace(state.publicURL)
	profile.ReadOnly = state.readOnly
	if profile.CredentialSource != "cloudflare" {
		profile.CredentialSource = state.credentialSource
	}
	if profile.Name == "" {
		state.err = core.Invalid("Enter a connection name")
		return
	}
	state.testing = true
	state.err = nil
	state.notice = ""
	input := service.ProfileInput{
		Profile: profile, AccessKeyID: state.accessKeyID, SecretAccessKey: state.secretKey,
	}
	go func() {
		buckets, err := a.svc.TestDraft(a.ctx, input)
		a.update(func() {
			state.testing = false
			if err != nil {
				state.err = err
				return
			}
			state.notice = a.tn("connection.testOk", len(buckets), "count", len(buckets))
		})
	}()
}

// askRemoveProfile confirms before a connection is removed.
func (a *App) askRemoveProfile() {
	state := &a.connect
	id := state.editing
	name := state.name
	a.confirm = confirmState{
		open:        true,
		title:       a.tf("connection.deleteTitle", "name", name),
		message:     a.t("connection.deleteDesc"),
		confirm:     a.t("connection.delete"),
		destructive: true,
		action: func() {
			go func() {
				err := a.svc.RemoveProfile(a.ctx, id, true)
				a.update(func() {
					if err != nil {
						a.toastCurrent(a.errorText(err))
						return
					}
					a.closeConnect()
					a.forgetProfile(id)
					a.loadProfiles()
					a.toastCurrent(a.t("connection.deleted"))
				})
			}()
		},
	}
}

// beginCloudflareSignIn starts the browser authorization.
func (a *App) beginCloudflareSignIn() {
	state := &a.connect
	state.signInBusy = true
	state.signInErr = nil
	state.accounts = nil
	state.buckets = nil
	a.svc.OnCloudflareSignIn = func(update service.CloudflareSignIn) {
		a.update(func() {
			if !a.connect.open {
				return
			}
			a.connect.signIn = update
			if update.Error != nil {
				a.connect.signInBusy = false
				a.connect.signInErr = update.Error
				return
			}
			if len(update.Accounts) > 0 {
				a.connect.accounts = update.Accounts
				if a.connect.accountID2 == "" {
					a.connect.accountID2 = update.Accounts[0].ID
					a.loadCloudflareBuckets()
				}
			}
		})
	}
	go func() {
		session, err := a.svc.StartCloudflareSignIn(a.ctx)
		a.update(func() {
			state.signIn = session
			if err != nil {
				state.signInBusy = false
				state.signInErr = err
				return
			}
			a.openExternally(session.AuthURL)
		})
	}()
}

// loadCloudflareBuckets reads the buckets of the authorized account.
func (a *App) loadCloudflareBuckets() {
	state := &a.connect
	if state.signIn.SessionID == "" || state.accountID2 == "" {
		return
	}
	session, account := state.signIn.SessionID, state.accountID2
	go func() {
		buckets, err := a.svc.CloudflareBuckets(a.ctx, session, account)
		a.update(func() {
			if a.connect.signIn.SessionID != session || a.connect.accountID2 != account {
				return
			}
			if err != nil {
				a.connect.signInErr = err
				return
			}
			a.connect.buckets = buckets
			if a.connect.bucket2 == "" && len(buckets) > 0 {
				a.connect.bucket2 = buckets[0].Name
			}
			a.connect.step = stepBucket
		})
	}()
}

// cancelSignIn stops a pending authorization.
func (a *App) cancelSignIn() {
	state := &a.connect
	session := state.signIn.SessionID
	state.signInBusy = false
	state.signIn = service.CloudflareSignIn{}
	go func() {
		if session != "" {
			_ = a.svc.CancelCloudflareSignIn(a.ctx, session)
		}
	}()
}

// connectCloudflare saves a connection from the authorization.
func (a *App) connectCloudflare() {
	state := &a.connect
	bucket := state.bucket2
	if bucket == "" && len(state.buckets) > 0 {
		bucket = state.buckets[0].Name
	}
	if bucket == "" {
		state.signInErr = core.Invalid("Choose a bucket")
		return
	}
	name := strings.TrimSpace(state.name)
	if name == "" {
		name = bucket
	}
	input := service.CloudflareConnection{
		SessionID: state.signIn.SessionID,
		ProfileID: state.editing,
		Name:      name,
		AccountID: state.accountID2,
		Bucket:    bucket,
		PublicURL: strings.TrimSpace(state.publicURL),
		ReadOnly:  state.readOnly2,
	}
	state.busy = true
	go func() {
		profile, err := a.svc.ConnectCloudflare(a.ctx, input)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.signInErr = err
				return
			}
			a.closeConnect()
			a.loadProfiles()
			a.toastCurrent(a.tf("connection.cloudflare.connected", "bucket", profile.Bucket))
		})
	}()
}

// loadCloudflareStatus reads the stored authorization of a connection.
func (a *App) loadCloudflareStatus(profileID string) {
	go func() {
		status, err := a.svc.CloudflareStatus(a.ctx, profileID)
		if err != nil {
			return
		}
		a.update(func() {
			if a.connect.editing == profileID {
				a.connect.status = status
			}
		})
	}()
}

// reauthorizeCloudflare renews the R2 token of a connection.
func (a *App) reauthorizeCloudflare() {
	state := &a.connect
	id := state.editing
	state.busy = true
	go func() {
		_, err := a.svc.ReauthorizeCloudflare(a.ctx, id)
		a.update(func() {
			state.busy = false
			if err != nil {
				state.err = err
				return
			}
			a.toastCurrent(a.t("connection.cloudflare.reauthorized"))
		})
	}()
}

// askDisconnectCloudflare confirms before the R2 token is deleted.
func (a *App) askDisconnectCloudflare() {
	state := &a.connect
	id := state.editing
	name := state.name
	a.confirm = confirmState{
		open:        true,
		title:       a.tf("connection.cloudflare.disconnectTitle", "name", name),
		message:     a.t("connection.cloudflare.disconnectDesc"),
		confirm:     a.t("connection.cloudflare.disconnect"),
		destructive: true,
		action: func() {
			go func() {
				err := a.svc.DisconnectCloudflare(a.ctx, id, true)
				a.update(func() {
					if err != nil {
						a.toastCurrent(a.errorText(err))
						return
					}
					a.closeConnect()
					a.loadProfiles()
					a.toastCurrent(a.t("connection.cloudflare.disconnected"))
				})
			}()
		},
	}
}
