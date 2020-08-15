package main

type Metadata struct{
	Name string `json:"name"`
	Description string `json:"description"`
}

type Config struct{
	Order []string `json:"order"`
	Jellyfin struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Username struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"username"`
		} `json:"username" cfg:"username"`
		Password struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"password"`
		} `json:"password" cfg:"password"`
		Server struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"server"`
		} `json:"server" cfg:"server"`
		PublicServer struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"public_server"`
		} `json:"public_server" cfg:"public_server"`
		Client struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"client"`
		} `json:"client" cfg:"client"`
		Version struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"version"`
		} `json:"version" cfg:"version"`
		Device struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"device"`
		} `json:"device" cfg:"device"`
		DeviceId struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"device_id"`
		} `json:"device_id" cfg:"device_id"`
	} `json:"jellyfin"`
	Ui struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Theme struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Options []string `json:"options"`
			Value string `json:"value" cfg:"theme"`
		} `json:"theme" cfg:"theme"`
		Host struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"host"`
		} `json:"host" cfg:"host"`
		Port struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value int `json:"value" cfg:"port"`
		} `json:"port" cfg:"port"`
		JellyfinLogin struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"jellyfin_login"`
		} `json:"jellyfin_login" cfg:"jellyfin_login"`
		AdminOnly struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"admin_only"`
		} `json:"admin_only" cfg:"admin_only"`
		Username struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"username"`
		} `json:"username" cfg:"username"`
		Password struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"password"`
		} `json:"password" cfg:"password"`
		Email struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"email"`
		} `json:"email" cfg:"email"`
		Debug struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"debug"`
		} `json:"debug" cfg:"debug"`
		ContactMessage struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"contact_message"`
		} `json:"contact_message" cfg:"contact_message"`
		HelpMessage struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"help_message"`
		} `json:"help_message" cfg:"help_message"`
		SuccessMessage struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"success_message"`
		} `json:"success_message" cfg:"success_message"`
		Bs5 struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"bs5"`
		} `json:"bs5" cfg:"bs5"`
	} `json:"ui"`
	PasswordValidation struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Enabled struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"enabled"`
		} `json:"enabled" cfg:"enabled"`
		MinLength struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"min_length"`
		} `json:"min_length" cfg:"min_length"`
		Upper struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"upper"`
		} `json:"upper" cfg:"upper"`
		Lower struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"lower"`
		} `json:"lower" cfg:"lower"`
		Number struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"number"`
		} `json:"number" cfg:"number"`
		Special struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"special"`
		} `json:"special" cfg:"special"`
	} `json:"password_validation"`
	Email struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		NoUsername struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"no_username"`
		} `json:"no_username" cfg:"no_username"`
		Use24H struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"use_24h"`
		} `json:"use_24h" cfg:"use_24h"`
		DateFormat struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"date_format"`
		} `json:"date_format" cfg:"date_format"`
		Message struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"message"`
		} `json:"message" cfg:"message"`
		Method struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Options []string `json:"options"`
			Value string `json:"value" cfg:"method"`
		} `json:"method" cfg:"method"`
		Address struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"address"`
		} `json:"address" cfg:"address"`
		From struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"from"`
		} `json:"from" cfg:"from"`
	} `json:"email"`
	PasswordResets struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Enabled struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"enabled"`
		} `json:"enabled" cfg:"enabled"`
		WatchDirectory struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"watch_directory"`
		} `json:"watch_directory" cfg:"watch_directory"`
		EmailHtml struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"email_html"`
		} `json:"email_html" cfg:"email_html"`
		EmailText struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"email_text"`
		} `json:"email_text" cfg:"email_text"`
		Subject struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"subject"`
		} `json:"subject" cfg:"subject"`
	} `json:"password_resets"`
	InviteEmails struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Enabled struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"enabled"`
		} `json:"enabled" cfg:"enabled"`
		EmailHtml struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"email_html"`
		} `json:"email_html" cfg:"email_html"`
		EmailText struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"email_text"`
		} `json:"email_text" cfg:"email_text"`
		Subject struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"subject"`
		} `json:"subject" cfg:"subject"`
		UrlBase struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"url_base"`
		} `json:"url_base" cfg:"url_base"`
	} `json:"invite_emails"`
	Notifications struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Enabled struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value bool `json:"value" cfg:"enabled"`
		} `json:"enabled" cfg:"enabled"`
		ExpiryHtml struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"expiry_html"`
		} `json:"expiry_html" cfg:"expiry_html"`
		ExpiryText struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"expiry_text"`
		} `json:"expiry_text" cfg:"expiry_text"`
		CreatedHtml struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"created_html"`
		} `json:"created_html" cfg:"created_html"`
		CreatedText struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"created_text"`
		} `json:"created_text" cfg:"created_text"`
	} `json:"notifications"`
	Mailgun struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		ApiUrl struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"api_url"`
		} `json:"api_url" cfg:"api_url"`
		ApiKey struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"api_key"`
		} `json:"api_key" cfg:"api_key"`
	} `json:"mailgun"`
	Smtp struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Encryption struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Options []string `json:"options"`
			Value string `json:"value" cfg:"encryption"`
		} `json:"encryption" cfg:"encryption"`
		Server struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"server"`
		} `json:"server" cfg:"server"`
		Port struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value int `json:"value" cfg:"port"`
		} `json:"port" cfg:"port"`
		Password struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"password"`
		} `json:"password" cfg:"password"`
	} `json:"smtp"`
	Files struct{
		Order []string `json:"order"`
		Meta Metadata `json:"meta"`
		Invites struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"invites"`
		} `json:"invites" cfg:"invites"`
		Emails struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"emails"`
		} `json:"emails" cfg:"emails"`
		UserTemplate struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"user_template"`
		} `json:"user_template" cfg:"user_template"`
		UserConfiguration struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"user_configuration"`
		} `json:"user_configuration" cfg:"user_configuration"`
		UserDisplayprefs struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"user_displayprefs"`
		} `json:"user_displayprefs" cfg:"user_displayprefs"`
		CustomCss struct{
			Name string `json:"name"`
			Required bool `json:"required"`
			Restart bool `json:"requires_restart"`
			Description string `json:"description"`
			Type string `json:"type"`
			Value string `json:"value" cfg:"custom_css"`
		} `json:"custom_css" cfg:"custom_css"`
	} `json:"files"`
}
